package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/nlp"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type ArenaService struct {
	challengeRepo   *repository.ChallengeRepository
	submissionRepo  *repository.SubmissionRepository
	userRepo        *repository.UserRepository
	evaluator       nlp.SemanticMatcher
	achievementSvc  *AchievementService
	notificationSvc *NotificationService
	logger          *slog.Logger
}

func NewArenaService(
	challengeRepo *repository.ChallengeRepository,
	submissionRepo *repository.SubmissionRepository,
	userRepo *repository.UserRepository,
	evaluator nlp.SemanticMatcher,
	achievementSvc *AchievementService,
	notificationSvc *NotificationService,
	logger *slog.Logger,
) *ArenaService {
	return &ArenaService{
		challengeRepo:   challengeRepo,
		submissionRepo:  submissionRepo,
		userRepo:        userRepo,
		evaluator:       evaluator,
		achievementSvc:  achievementSvc,
		notificationSvc: notificationSvc,
		logger:          logger,
	}
}

func (s *ArenaService) ListChallenges(ctx context.Context, filter model.ChallengeFilter) ([]model.ChallengeListItem, int, error) {
	return s.challengeRepo.List(ctx, filter)
}

// GetChallenge returns a challenge with sensitive fields (target_vulnerability, conceptual_fix)
// stripped out. Those fields are only used server-side for evaluation.
func (s *ArenaService) GetChallenge(ctx context.Context, id uuid.UUID) (*model.Challenge, error) {
	ch, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Strip solution data — never sent to client
	ch.TargetVulnerability = ""
	ch.ConceptualFix = ""
	ch.VulnerableLines = nil

	return ch, nil
}

type SubmitAnswerInput struct {
	AnswerText   string `json:"answer_text"`
	TargetLines  []int  `json:"target_lines,omitempty"`
	TimeSpentSec *int   `json:"time_spent_sec,omitempty"`
}

type SubmitAnswerResult struct {
	Submission *model.Submission              `json:"submission"`
	Feedback   *model.EvaluationFeedback      `json:"feedback"`
	Progress   *model.UserChallengeProgress   `json:"progress"`
	FirstBlood bool                           `json:"first_blood"`
	BonusXP    int                            `json:"bonus_xp,omitempty"`
}

func (s *ArenaService) SubmitAnswer(ctx context.Context, userID, challengeID uuid.UUID, input SubmitAnswerInput) (*SubmitAnswerResult, error) {
	// Fetch challenge with solution data for evaluation
	ch, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("challenge not found")
		}
		return nil, err
	}

	// Run semantic evaluation
	evalResult, err := s.evaluator.Evaluate(ctx, nlp.EvaluationRequest{
		UserAnswer:          input.AnswerText,
		TargetVulnerability: ch.TargetVulnerability,
		ConceptualFix:       ch.ConceptualFix,
		Language:            ch.Language.Slug,
		Difficulty:          ch.Difficulty,
		VulnerableLines:     ch.VulnerableLines,
		UserTargetLines:     input.TargetLines,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	// Build feedback
	feedback := &model.EvaluationFeedback{
		VulnIdentified:   evalResult.VulnIdentified,
		VulnScore:        evalResult.VulnScore,
		FixIdentified:    evalResult.FixIdentified,
		FixScore:         evalResult.FixScore,
		LineAccuracy:     evalResult.LineAccuracy,
		OverallScore:     evalResult.OverallScore,
		Passed:           evalResult.Passed,
		TerminalLog:      evalResult.TerminalLog,
		MatchedVulnTerms: evalResult.MatchedVulnTerms,
		MatchedFixTerms:  evalResult.MatchedFixTerms,
	}

	feedbackJSON, err := json.Marshal(feedback)
	if err != nil {
		return nil, fmt.Errorf("marshaling feedback: %w", err)
	}

	// First blood detection: if the user passed, try to claim first blood
	isFirstBlood := false
	bonusXP := 0
	if evalResult.Passed && ch.FirstBloodUserID == nil {
		claimed, fbErr := s.challengeRepo.ClaimFirstBlood(ctx, challengeID, userID)
		if fbErr == nil && claimed {
			isFirstBlood = true
			bonusXP = ch.Points / 4 // 25% XP bonus

			username := s.resolveUsername(ctx, userID)

			// Audit log: first blood is a critical platform event
			s.logger.Info("audit_event",
				slog.String("log_type", "audit"),
				slog.String("category", "first_blood"),
				slog.String("user_id", userID.String()),
				slog.String("username", username),
				slog.String("challenge_id", challengeID.String()),
				slog.String("challenge_title", ch.Title),
				slog.Int("bonus_xp", bonusXP),
			)

			// Fire Discord webhook for First Blood (async)
			s.notificationSvc.SendFirstBlood(ctx, username, ch.Title, bonusXP)
		}
	}

	// Create submission record
	sub := &model.Submission{
		ID:           uuid.New(),
		UserID:       userID,
		ChallengeID:  challengeID,
		AnswerText:   input.AnswerText,
		TargetLines:  input.TargetLines,
		Score:        evalResult.OverallScore,
		IsCorrect:    evalResult.Passed,
		IsFirstBlood: isFirstBlood,
		Feedback:     feedbackJSON,
		TimeSpentSec: input.TimeSpentSec,
	}

	if err := s.submissionRepo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("saving submission: %w", err)
	}

	// Update progress (use boosted points for first blood)
	effectiveScore := evalResult.OverallScore
	if err := s.submissionRepo.UpsertProgress(ctx, userID, challengeID, effectiveScore, evalResult.Passed); err != nil {
		return nil, fmt.Errorf("updating progress: %w", err)
	}

	progress, err := s.submissionRepo.GetProgress(ctx, userID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("getting progress: %w", err)
	}

	// Fire achievement checks asynchronously if the user passed
	if evalResult.Passed {
		username := s.resolveUsername(ctx, userID)
		go s.achievementSvc.CheckAndGrant(context.Background(), AchievementEvent{
			UserID:        userID,
			Username:      username,
			IsFirstBlood:  isFirstBlood,
			ChallengeName: ch.Title,
			BonusXP:       bonusXP,
			SubmittedAt:   time.Now().UTC(),
		})
	}

	return &SubmitAnswerResult{
		Submission: sub,
		Feedback:   feedback,
		Progress:   progress,
		FirstBlood: isFirstBlood,
		BonusXP:    bonusXP,
	}, nil
}

// resolveUsername fetches the username for a user ID (best-effort for async notifications).
func (s *ArenaService) resolveUsername(ctx context.Context, userID uuid.UUID) string {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "unknown"
	}
	return user.Username
}

func (s *ArenaService) GetUserProgress(ctx context.Context, userID, challengeID uuid.UUID) (*model.UserChallengeProgress, error) {
	return s.submissionRepo.GetProgress(ctx, userID, challengeID)
}

func (s *ArenaService) GetSubmissionHistory(ctx context.Context, userID, challengeID uuid.UUID) ([]model.Submission, error) {
	return s.submissionRepo.ListByUserAndChallenge(ctx, userID, challengeID)
}

// RevealSolutionResult contains the solution data plus a 0-score submission record.
type RevealSolutionResult struct {
	Solution   *model.RevealResult            `json:"solution"`
	Submission *model.Submission              `json:"submission"`
	Progress   *model.UserChallengeProgress   `json:"progress"`
}

// RevealSolution exposes the challenge's solution to the user.
// It creates a 0-score submission to record the reveal and prevent leaderboard gaming.
func (s *ArenaService) RevealSolution(ctx context.Context, userID, challengeID uuid.UUID) (*RevealSolutionResult, error) {
	ch, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("challenge not found: %w", err)
	}

	// Build revealed feedback
	feedback := &model.EvaluationFeedback{
		IsRevealed: true,
		TerminalLog: []string{
			"> Solution revealed by operator.",
			"> Points for this challenge set to 0.",
			"> STATUS: REVEALED — No points awarded.",
		},
	}

	feedbackJSON, err := json.Marshal(feedback)
	if err != nil {
		return nil, fmt.Errorf("marshaling feedback: %w", err)
	}

	sub := &model.Submission{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		AnswerText:  "[Solution Revealed]",
		Score:       0,
		IsCorrect:   false,
		Feedback:    feedbackJSON,
	}

	if err := s.submissionRepo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("saving reveal submission: %w", err)
	}

	// Update progress with 0 score (won't overwrite a better existing score)
	if err := s.submissionRepo.UpsertProgress(ctx, userID, challengeID, 0, false); err != nil {
		return nil, fmt.Errorf("updating progress: %w", err)
	}

	progress, err := s.submissionRepo.GetProgress(ctx, userID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("getting progress: %w", err)
	}

	s.logger.Info("audit_event",
		slog.String("log_type", "audit"),
		slog.String("category", "solution_revealed"),
		slog.String("user_id", userID.String()),
		slog.String("challenge_id", challengeID.String()),
		slog.String("challenge_title", ch.Title),
	)

	return &RevealSolutionResult{
		Solution: &model.RevealResult{
			TargetVulnerability: ch.TargetVulnerability,
			ConceptualFix:       ch.ConceptualFix,
			VulnerableLines:     ch.VulnerableLines,
		},
		Submission: sub,
		Progress:   progress,
	}, nil
}
