package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type AchievementService struct {
	achievementRepo *repository.AchievementRepository
	notificationSvc *NotificationService
	logger          *slog.Logger
}

func NewAchievementService(
	achievementRepo *repository.AchievementRepository,
	notificationSvc *NotificationService,
	logger *slog.Logger,
) *AchievementService {
	return &AchievementService{
		achievementRepo: achievementRepo,
		notificationSvc: notificationSvc,
		logger:          logger,
	}
}

// AchievementEvent carries context about what just happened, for rule evaluation.
type AchievementEvent struct {
	UserID         uuid.UUID
	Username       string
	IsFirstBlood   bool
	ChallengeName  string
	BonusXP        int
	SubmittedAt    time.Time
}

// CheckAndGrant evaluates all achievement rules against the event and grants any newly earned badges.
// Designed to be called asynchronously (in a goroutine) so it never blocks submission flow.
func (s *AchievementService) CheckAndGrant(ctx context.Context, event AchievementEvent) {
	// Rule 1: First Blood Spiller — triggered by any first blood
	if event.IsFirstBlood {
		s.tryGrant(ctx, event.UserID, event.Username, "first-blood-spiller")
	}

	// Rule 2: SQLi Master — solved 3+ injection-category challenges
	injectionCount, err := s.achievementRepo.GetSolvedCountByCategory(ctx, event.UserID, "injection")
	if err != nil {
		s.logger.Error("achievement check failed", slog.String("rule", "sqli-master"), slog.String("error", err.Error()))
	} else if injectionCount >= 3 {
		s.tryGrant(ctx, event.UserID, event.Username, "sqli-master")
	}

	// Rule 3: Night Owl — submission between 00:00 and 05:00 UTC
	hour := event.SubmittedAt.UTC().Hour()
	if hour >= 0 && hour < 5 {
		s.tryGrant(ctx, event.UserID, event.Username, "night-owl")
	}

	// Rule 4: Persistence — 7-day streak
	streak, err := s.achievementRepo.GetCurrentStreak(ctx, event.UserID)
	if err != nil {
		s.logger.Error("achievement check failed", slog.String("rule", "persistence"), slog.String("error", err.Error()))
	} else if streak >= 7 {
		s.tryGrant(ctx, event.UserID, event.Username, "persistence")
	}

	// Rule 5: Pentester — 10+ total challenges solved
	totalSolved, err := s.achievementRepo.GetTotalSolved(ctx, event.UserID)
	if err != nil {
		s.logger.Error("achievement check failed", slog.String("rule", "pentester"), slog.String("error", err.Error()))
	} else if totalSolved >= 10 {
		s.tryGrant(ctx, event.UserID, event.Username, "pentester")
	}
}

func (s *AchievementService) tryGrant(ctx context.Context, userID uuid.UUID, username, slug string) {
	ua, granted, err := s.achievementRepo.Grant(ctx, userID, slug)
	if err != nil {
		s.logger.Error("failed to grant achievement",
			slog.String("slug", slug),
			slog.String("error", err.Error()),
		)
		return
	}
	if granted && ua != nil {
		s.logger.Info("achievement unlocked",
			slog.String("user", username),
			slog.String("achievement", ua.Achievement.Name),
		)
		s.notificationSvc.SendAchievementUnlocked(ctx, username, ua.Achievement.Name)
	}
}

// GetUserAchievements returns all badges unlocked by a user.
func (s *AchievementService) GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]model.UserAchievement, error) {
	return s.achievementRepo.GetUserAchievements(ctx, userID)
}

// GetAllAchievements returns the full badge catalog.
func (s *AchievementService) GetAllAchievements(ctx context.Context) ([]model.Achievement, error) {
	return s.achievementRepo.ListAll(ctx)
}
