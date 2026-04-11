package model

import "time"

// DashboardProfile is the complete response for GET /dashboard/profile.
type DashboardProfile struct {
	Rank           RankInfo           `json:"rank"`
	Stats          OverallStats       `json:"stats"`
	Achievements   []UserAchievement  `json:"achievements"`
	SkillRadar     []SkillRadarPoint  `json:"skill_radar"`
	RecentActivity []ActivityEntry    `json:"recent_activity"`
	NextChallenge  *ChallengeListItem `json:"next_challenge,omitempty"`
}

// RankInfo describes the user's current rank/tier.
type RankInfo struct {
	Title      string  `json:"title"`
	Tier       int     `json:"tier"`       // 1-6
	TotalXP    int     `json:"total_xp"`
	NextTierXP int     `json:"next_tier_xp"` // XP needed for next rank
	Progress   float64 `json:"progress"`     // 0.0-1.0 within current tier
}

type OverallStats struct {
	TotalSolved     int     `json:"total_solved"`
	TotalAvailable  int     `json:"total_available"`
	TotalAttempted  int     `json:"total_attempted"`
	TotalPoints     int     `json:"total_points"`
	AverageScore    float64 `json:"average_score"`
	LessonsRead     int     `json:"lessons_read"`
	CurrentStreak   int     `json:"current_streak"`
}

// SkillRadarPoint is one axis of the radar chart.
type SkillRadarPoint struct {
	Category string  `json:"category"`
	Slug     string  `json:"slug"`
	Score    float64 `json:"score"`    // 0-100 proficiency
	Solved   int     `json:"solved"`
	Total    int     `json:"total"`
}

// ActivityEntry is a single item in the recent activity feed.
type ActivityEntry struct {
	Type       string    `json:"type"` // "challenge_solved", "challenge_attempted", "lesson_completed"
	Title      string    `json:"title"`
	Points     int       `json:"points,omitempty"`
	Score      float64   `json:"score,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// LeaderboardEntry represents one row in the global leaderboard.
type LeaderboardEntry struct {
	Rank        int    `json:"rank"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	RankTitle   string `json:"rank_title"`
	Tier        int    `json:"tier"`
	TotalXP     int    `json:"total_xp"`
	TotalSolved int    `json:"total_solved"`
}

// PublicProfile is the public-facing profile for any user (looked up by username).
type PublicProfile struct {
	Username       string             `json:"username"`
	DisplayName    string             `json:"display_name"`
	AvatarURL      string             `json:"avatar_url"`
	JoinedAt       time.Time          `json:"joined_at"`
	Rank           RankInfo           `json:"rank"`
	Stats          PublicStats        `json:"stats"`
	Achievements   []UserAchievement  `json:"achievements"`
	SkillRadar     []SkillRadarPoint  `json:"skill_radar"`
	RecentActivity []ActivityEntry    `json:"recent_activity"`
}

// PublicStats is a privacy-safe subset of OverallStats.
type PublicStats struct {
	TotalSolved int `json:"total_solved"`
	TotalPoints int `json:"total_points"`
}

// Rank tiers (XP thresholds)
var RankTiers = []struct {
	Title    string
	MinXP    int
}{
	{"Script Kiddie", 0},
	{"Hacker", 200},
	{"Pro Hacker", 600},
	{"Elite Hacker", 1200},
	{"Guru", 2500},
	{"Omniscient", 5000},
}
