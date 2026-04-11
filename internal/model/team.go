package model

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Tag         string    `json:"tag"`
	Description string    `json:"description"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMember struct {
	TeamID      uuid.UUID `json:"team_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"` // "leader" | "member"
	JoinedAt    time.Time `json:"joined_at"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
}

type TeamWithMembers struct {
	Team        Team         `json:"team"`
	Members     []TeamMember `json:"members"`
	TotalXP     int          `json:"total_xp"`
	TotalSolved int          `json:"total_solved"`
}

type TeamLeaderboardEntry struct {
	Rank        int    `json:"rank"`
	TeamName    string `json:"team_name"`
	Tag         string `json:"tag"`
	MemberCount int    `json:"member_count"`
	TotalXP     int    `json:"total_xp"`
	TotalSolved int    `json:"total_solved"`
}
