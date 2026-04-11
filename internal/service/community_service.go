package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type CommunityService struct {
	communityRepo *repository.CommunityRepository
	telemetryRepo *repository.TelemetryRepository
	logger        *slog.Logger
}

func NewCommunityService(
	communityRepo *repository.CommunityRepository,
	telemetryRepo *repository.TelemetryRepository,
	logger *slog.Logger,
) *CommunityService {
	return &CommunityService{
		communityRepo: communityRepo,
		telemetryRepo: telemetryRepo,
		logger:        logger,
	}
}

type CommunitySubmitInput struct {
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Difficulty          int      `json:"difficulty"`
	LanguageSlug        string   `json:"language_slug"`
	VulnCategorySlug    string   `json:"vuln_category_slug"`
	VulnerableCode      string   `json:"vulnerable_code"`
	TargetVulnerability string   `json:"target_vulnerability"`
	ConceptualFix       string   `json:"conceptual_fix"`
	VulnerableLines     string   `json:"vulnerable_lines"`
	Hints               []string `json:"hints"`
	Points              int      `json:"points"`
}

// SubmitChallenge creates a new community challenge submission after verifying XP threshold.
func (s *CommunityService) SubmitChallenge(ctx context.Context, userID uuid.UUID, input CommunitySubmitInput) (*model.CommunityChallenge, error) {
	// Verify user meets XP threshold
	stats, err := s.telemetryRepo.GetOverallStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checking user stats: %w", err)
	}
	if stats.TotalPoints < model.MinXPForCommunitySubmission {
		return nil, fmt.Errorf("insufficient XP: need %d, have %d (Pro Hacker rank required)",
			model.MinXPForCommunitySubmission, stats.TotalPoints)
	}

	if input.Points <= 0 {
		input.Points = 100
	}
	if input.Hints == nil {
		input.Hints = []string{}
	}

	ch := &model.CommunityChallenge{
		ID:                  uuid.New(),
		AuthorID:            userID,
		Title:               input.Title,
		Description:         input.Description,
		Difficulty:          input.Difficulty,
		LanguageSlug:        input.LanguageSlug,
		VulnCategorySlug:    input.VulnCategorySlug,
		VulnerableCode:      input.VulnerableCode,
		TargetVulnerability: input.TargetVulnerability,
		ConceptualFix:       input.ConceptualFix,
		VulnerableLines:     input.VulnerableLines,
		Hints:               input.Hints,
		Points:              input.Points,
		Status:              model.CommunityStatusPending,
	}

	if err := s.communityRepo.Insert(ctx, ch); err != nil {
		return nil, fmt.Errorf("creating community challenge: %w", err)
	}

	s.logger.Info("community challenge submitted",
		slog.String("id", ch.ID.String()),
		slog.String("author", userID.String()),
		slog.String("title", ch.Title))

	return ch, nil
}

// ListMyChallenges returns all community challenges submitted by the user.
func (s *CommunityService) ListMyChallenges(ctx context.Context, userID uuid.UUID) ([]model.CommunityChallenge, int, error) {
	return s.communityRepo.ListByAuthor(ctx, userID, 50, 0)
}

// GetMyChallenge returns a community challenge if it belongs to the user.
func (s *CommunityService) GetMyChallenge(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.CommunityChallenge, error) {
	ch, err := s.communityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.AuthorID != userID {
		return nil, fmt.Errorf("not found")
	}
	return ch, nil
}

// UpdateChallenge updates a pending community challenge.
func (s *CommunityService) UpdateChallenge(ctx context.Context, userID uuid.UUID, id uuid.UUID, input CommunitySubmitInput) (*model.CommunityChallenge, error) {
	ch, err := s.communityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.AuthorID != userID {
		return nil, fmt.Errorf("not found")
	}
	if ch.Status != model.CommunityStatusPending {
		return nil, fmt.Errorf("can only edit pending submissions")
	}

	if input.Hints == nil {
		input.Hints = []string{}
	}
	if input.Points <= 0 {
		input.Points = 100
	}

	ch.Title = input.Title
	ch.Description = input.Description
	ch.Difficulty = input.Difficulty
	ch.LanguageSlug = input.LanguageSlug
	ch.VulnCategorySlug = input.VulnCategorySlug
	ch.VulnerableCode = input.VulnerableCode
	ch.TargetVulnerability = input.TargetVulnerability
	ch.ConceptualFix = input.ConceptualFix
	ch.VulnerableLines = input.VulnerableLines
	ch.Hints = input.Hints
	ch.Points = input.Points

	if err := s.communityRepo.Update(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// DeleteChallenge deletes a pending community challenge.
func (s *CommunityService) DeleteChallenge(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	ch, err := s.communityRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ch.AuthorID != userID {
		return fmt.Errorf("not found")
	}
	return s.communityRepo.Delete(ctx, id)
}

// GetUserXP returns the user's current XP for frontend display (e.g., to check eligibility).
func (s *CommunityService) GetUserXP(ctx context.Context, userID uuid.UUID) (int, error) {
	stats, err := s.telemetryRepo.GetOverallStats(ctx, userID)
	if err != nil {
		return 0, err
	}
	return stats.TotalPoints, nil
}
