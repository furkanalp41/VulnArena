package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Username        string     `json:"username"`
	PasswordHash    string     `json:"-"`
	DisplayName     string     `json:"display_name,omitempty"`
	AvatarURL       string     `json:"avatar_url,omitempty"`
	Role            string     `json:"role"`
	ApiKey          *string    `json:"-"`
	ApiKeyHint      *string    `json:"api_key_hint,omitempty"`
	ApiKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

const (
	RoleUser           = "user"
	RoleAdmin          = "admin"
	RoleContentCreator = "content_creator"
)
