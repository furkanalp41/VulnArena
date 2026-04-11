package model

import (
	"time"

	"github.com/google/uuid"
)

// Achievement represents a collectible badge in the platform.
type Achievement struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IconSVG     string    `json:"icon_svg"`
	Category    string    `json:"category"` // "combat", "dedication", "mastery", "special"
	XPReward    int       `json:"xp_reward"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserAchievement is a badge unlocked by a specific user.
type UserAchievement struct {
	Achievement Achievement `json:"achievement"`
	UnlockedAt  time.Time   `json:"unlocked_at"`
}
