package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type TelemetryService struct {
	telemetryRepo   *repository.TelemetryRepository
	userRepo        *repository.UserRepository
	achievementRepo *repository.AchievementRepository
	redisClient     *redis.Client
}

func NewTelemetryService(
	telemetryRepo *repository.TelemetryRepository,
	userRepo *repository.UserRepository,
	achievementRepo *repository.AchievementRepository,
	redisClient *redis.Client,
) *TelemetryService {
	return &TelemetryService{
		telemetryRepo:   telemetryRepo,
		userRepo:        userRepo,
		achievementRepo: achievementRepo,
		redisClient:     redisClient,
	}
}

func (s *TelemetryService) GetDashboardProfile(ctx context.Context, userID uuid.UUID) (*model.DashboardProfile, error) {
	stats, err := s.telemetryRepo.GetOverallStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	radar, err := s.telemetryRepo.GetSkillRadar(ctx, userID)
	if err != nil {
		return nil, err
	}

	activity, err := s.telemetryRepo.GetRecentActivity(ctx, userID, 15)
	if err != nil {
		return nil, err
	}

	next, err := s.telemetryRepo.GetNextChallenge(ctx, userID)
	if err != nil {
		return nil, err
	}

	achievements, err := s.achievementRepo.GetUserAchievements(ctx, userID)
	if err != nil {
		return nil, err
	}

	rank := computeRank(stats.TotalPoints)

	return &model.DashboardProfile{
		Rank:           rank,
		Stats:          *stats,
		Achievements:   achievements,
		SkillRadar:     radar,
		RecentActivity: activity,
		NextChallenge:  next,
	}, nil
}

const leaderboardCacheKey = "vulnarena:leaderboard"
const leaderboardCacheTTL = 60 * time.Second

// GetLeaderboard returns the top 50 users by XP, cached in Redis.
func (s *TelemetryService) GetLeaderboard(ctx context.Context) ([]model.LeaderboardEntry, error) {
	// Try cache first
	cached, err := s.redisClient.Get(ctx, leaderboardCacheKey).Result()
	if err == nil {
		var entries []model.LeaderboardEntry
		if json.Unmarshal([]byte(cached), &entries) == nil {
			return entries, nil
		}
	}

	// Cache miss — query DB
	rows, err := s.telemetryRepo.GetLeaderboard(ctx, 50)
	if err != nil {
		return nil, err
	}

	entries := make([]model.LeaderboardEntry, len(rows))
	for i, row := range rows {
		rank := computeRank(row.TotalXP)
		entries[i] = model.LeaderboardEntry{
			Rank:        i + 1,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarURL,
			RankTitle:   rank.Title,
			Tier:        rank.Tier,
			TotalXP:     row.TotalXP,
			TotalSolved: row.TotalSolved,
		}
	}

	// Cache the result
	if data, err := json.Marshal(entries); err == nil {
		s.redisClient.Set(ctx, leaderboardCacheKey, data, leaderboardCacheTTL)
	}

	return entries, nil
}

// GetPublicProfile returns a privacy-safe public profile for a user.
func (s *TelemetryService) GetPublicProfile(ctx context.Context, username string) (*model.PublicProfile, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	stats, err := s.telemetryRepo.GetOverallStats(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	radar, err := s.telemetryRepo.GetSkillRadar(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	activity, err := s.telemetryRepo.GetPublicActivity(ctx, user.ID, 10)
	if err != nil {
		return nil, err
	}

	achievements, err := s.achievementRepo.GetUserAchievements(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	rank := computeRank(stats.TotalPoints)

	return &model.PublicProfile{
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		JoinedAt:    user.CreatedAt,
		Rank:        rank,
		Stats: model.PublicStats{
			TotalSolved: stats.TotalSolved,
			TotalPoints: stats.TotalPoints,
		},
		Achievements:   achievements,
		SkillRadar:     radar,
		RecentActivity: activity,
	}, nil
}

func computeRank(xp int) model.RankInfo {
	tiers := model.RankTiers
	currentTier := 0
	currentTitle := tiers[0].Title

	for i, tier := range tiers {
		if xp >= tier.MinXP {
			currentTier = i
			currentTitle = tier.Title
		}
	}

	// Calculate progress within current tier
	var progress float64
	nextTierXP := 0
	if currentTier < len(tiers)-1 {
		currentMin := tiers[currentTier].MinXP
		nextMin := tiers[currentTier+1].MinXP
		nextTierXP = nextMin
		tierRange := nextMin - currentMin
		if tierRange > 0 {
			progress = float64(xp-currentMin) / float64(tierRange)
		}
	} else {
		progress = 1.0 // Max rank
		nextTierXP = tiers[currentTier].MinXP
	}

	return model.RankInfo{
		Title:      currentTitle,
		Tier:       currentTier + 1,
		TotalXP:    xp,
		NextTierXP: nextTierXP,
		Progress:   progress,
	}
}
