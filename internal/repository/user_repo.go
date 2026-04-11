package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("already exists")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.Username, user.PasswordHash, user.DisplayName, user.Role,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, username, password_hash, display_name, avatar_url, role,
		       api_key, api_key_hint, api_key_created_at, created_at, updated_at
		FROM users WHERE id = $1`

	return r.scanUser(ctx, query, id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, username, password_hash, display_name, avatar_url, role,
		       api_key, api_key_hint, api_key_created_at, created_at, updated_at
		FROM users WHERE email = $1`

	return r.scanUser(ctx, query, email)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, email, username, password_hash, display_name, avatar_url, role,
		       api_key, api_key_hint, api_key_created_at, created_at, updated_at
		FROM users WHERE username = $1`

	return r.scanUser(ctx, query, username)
}

func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKeyHash string) (*model.User, error) {
	query := `
		SELECT id, email, username, password_hash, display_name, avatar_url, role,
		       api_key, api_key_hint, api_key_created_at, created_at, updated_at
		FROM users WHERE api_key = $1`

	return r.scanUser(ctx, query, apiKeyHash)
}

func (r *UserRepository) SetAPIKey(ctx context.Context, userID uuid.UUID, hashedKey, hint string) error {
	query := `
		UPDATE users
		SET api_key = $2, api_key_hint = $3, api_key_created_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, userID, hashedKey, hint)
	if err != nil {
		return fmt.Errorf("setting api key: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) DeleteAPIKey(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET api_key = NULL, api_key_hint = NULL, api_key_created_at = NULL, updated_at = NOW()
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("deleting api key: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET display_name = $2, avatar_url = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, user.ID, user.DisplayName, user.AvatarURL).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

func (r *UserRepository) scanUser(ctx context.Context, query string, args ...any) (*model.User, error) {
	var u model.User
	var displayName, avatarURL *string

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&displayName, &avatarURL, &u.Role,
		&u.ApiKey, &u.ApiKeyHint, &u.ApiKeyCreatedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning user: %w", err)
	}

	if displayName != nil {
		u.DisplayName = *displayName
	}
	if avatarURL != nil {
		u.AvatarURL = *avatarURL
	}

	return &u, nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
