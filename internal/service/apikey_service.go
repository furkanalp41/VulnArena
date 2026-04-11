package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type APIKeyService struct {
	userRepo *repository.UserRepository
}

func NewAPIKeyService(userRepo *repository.UserRepository) *APIKeyService {
	return &APIKeyService{userRepo: userRepo}
}

type APIKeyInfo struct {
	Hint      string    `json:"hint"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateAPIKey creates a new API key for the user, replacing any existing one.
// Returns the raw key (shown only once). The database stores only the SHA-256 hash.
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate 20 random bytes → 40 hex chars
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	rawKey := "va_" + hex.EncodeToString(raw)
	hint := rawKey[len(rawKey)-4:]

	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	if err := s.userRepo.SetAPIKey(ctx, userID, hashedKey, hint); err != nil {
		return "", fmt.Errorf("storing api key: %w", err)
	}

	return rawKey, nil
}

// RevokeAPIKey removes the user's API key.
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.DeleteAPIKey(ctx, userID)
}

// GetAPIKeyInfo returns non-sensitive metadata about the user's API key.
func (s *APIKeyService) GetAPIKeyInfo(ctx context.Context, userID uuid.UUID) (*APIKeyInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.ApiKeyHint == nil || user.ApiKeyCreatedAt == nil {
		return nil, nil
	}

	return &APIKeyInfo{
		Hint:      *user.ApiKeyHint,
		CreatedAt: *user.ApiKeyCreatedAt,
	}, nil
}

// ValidateAPIKey validates a raw API key and returns the associated user.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, rawKey string) (*model.User, error) {
	if !strings.HasPrefix(rawKey, "va_") {
		return nil, fmt.Errorf("invalid api key format")
	}

	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	user, err := s.userRepo.GetByAPIKey(ctx, hashedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	return user, nil
}
