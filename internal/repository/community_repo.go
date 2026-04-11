package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type CommunityRepository struct {
	pool *pgxpool.Pool
}

func NewCommunityRepository(pool *pgxpool.Pool) *CommunityRepository {
	return &CommunityRepository{pool: pool}
}

func (r *CommunityRepository) Insert(ctx context.Context, ch *model.CommunityChallenge) error {
	hintsJSON, err := json.Marshal(ch.Hints)
	if err != nil {
		return fmt.Errorf("marshalling hints: %w", err)
	}

	query := `
		INSERT INTO community_challenges (
			id, author_id, title, description, difficulty,
			language_slug, vuln_category_slug, vulnerable_code,
			target_vulnerability, conceptual_fix, vulnerable_lines,
			hints, points, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		ch.ID, ch.AuthorID, ch.Title, ch.Description, ch.Difficulty,
		ch.LanguageSlug, ch.VulnCategorySlug, ch.VulnerableCode,
		ch.TargetVulnerability, ch.ConceptualFix, ch.VulnerableLines,
		hintsJSON, ch.Points, ch.Status,
	).Scan(&ch.CreatedAt, &ch.UpdatedAt)
}

func (r *CommunityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.CommunityChallenge, error) {
	query := `
		SELECT cc.id, cc.author_id, cc.title, cc.description, cc.difficulty,
			cc.language_slug, cc.vuln_category_slug, cc.vulnerable_code,
			cc.target_vulnerability, cc.conceptual_fix, cc.vulnerable_lines,
			cc.hints, cc.points, cc.status,
			cc.reviewer_id, cc.reviewer_notes, cc.challenge_id,
			cc.created_at, cc.updated_at,
			u.username
		FROM community_challenges cc
		JOIN users u ON u.id = cc.author_id
		WHERE cc.id = $1`

	return r.scanOne(ctx, query, id)
}

func (r *CommunityRepository) ListByAuthor(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]model.CommunityChallenge, int, error) {
	if limit <= 0 {
		limit = 20
	}

	countQuery := `SELECT COUNT(*) FROM community_challenges WHERE author_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, authorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting community challenges: %w", err)
	}

	query := `
		SELECT cc.id, cc.author_id, cc.title, cc.description, cc.difficulty,
			cc.language_slug, cc.vuln_category_slug, cc.vulnerable_code,
			cc.target_vulnerability, cc.conceptual_fix, cc.vulnerable_lines,
			cc.hints, cc.points, cc.status,
			cc.reviewer_id, cc.reviewer_notes, cc.challenge_id,
			cc.created_at, cc.updated_at,
			u.username
		FROM community_challenges cc
		JOIN users u ON u.id = cc.author_id
		WHERE cc.author_id = $1
		ORDER BY cc.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, authorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing community challenges: %w", err)
	}
	defer rows.Close()

	items, err := r.scanMany(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *CommunityRepository) ListByStatus(ctx context.Context, status string, limit, offset int) ([]model.CommunityChallenge, int, error) {
	if limit <= 0 {
		limit = 20
	}

	countQuery := `SELECT COUNT(*) FROM community_challenges WHERE status = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting community challenges: %w", err)
	}

	query := `
		SELECT cc.id, cc.author_id, cc.title, cc.description, cc.difficulty,
			cc.language_slug, cc.vuln_category_slug, cc.vulnerable_code,
			cc.target_vulnerability, cc.conceptual_fix, cc.vulnerable_lines,
			cc.hints, cc.points, cc.status,
			cc.reviewer_id, cc.reviewer_notes, cc.challenge_id,
			cc.created_at, cc.updated_at,
			u.username
		FROM community_challenges cc
		JOIN users u ON u.id = cc.author_id
		WHERE cc.status = $1
		ORDER BY cc.created_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing community challenges: %w", err)
	}
	defer rows.Close()

	items, err := r.scanMany(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *CommunityRepository) Update(ctx context.Context, ch *model.CommunityChallenge) error {
	hintsJSON, err := json.Marshal(ch.Hints)
	if err != nil {
		return fmt.Errorf("marshalling hints: %w", err)
	}

	query := `
		UPDATE community_challenges
		SET title = $2, description = $3, difficulty = $4,
			language_slug = $5, vuln_category_slug = $6, vulnerable_code = $7,
			target_vulnerability = $8, conceptual_fix = $9, vulnerable_lines = $10,
			hints = $11, points = $12, updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING updated_at`

	err = r.pool.QueryRow(ctx, query,
		ch.ID, ch.Title, ch.Description, ch.Difficulty,
		ch.LanguageSlug, ch.VulnCategorySlug, ch.VulnerableCode,
		ch.TargetVulnerability, ch.ConceptualFix, ch.VulnerableLines,
		hintsJSON, ch.Points,
	).Scan(&ch.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("updating community challenge: %w", err)
	}
	return nil
}

func (r *CommunityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM community_challenges WHERE id = $1 AND status = 'pending'`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting community challenge: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CommunityRepository) SetStatus(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, notes string) error {
	query := `
		UPDATE community_challenges
		SET status = $2, reviewer_id = $3, reviewer_notes = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id`

	var dummy uuid.UUID
	err := r.pool.QueryRow(ctx, query, id, status, reviewerID, notes).Scan(&dummy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("setting community challenge status: %w", err)
	}
	return nil
}

func (r *CommunityRepository) SetChallengeID(ctx context.Context, id uuid.UUID, challengeID uuid.UUID) error {
	query := `
		UPDATE community_challenges
		SET challenge_id = $2, status = 'published', updated_at = NOW()
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id, challengeID)
	if err != nil {
		return fmt.Errorf("linking published challenge: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanOne scans a single community challenge from a query.
func (r *CommunityRepository) scanOne(ctx context.Context, query string, args ...any) (*model.CommunityChallenge, error) {
	var ch model.CommunityChallenge
	var hintsJSON []byte

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&ch.ID, &ch.AuthorID, &ch.Title, &ch.Description, &ch.Difficulty,
		&ch.LanguageSlug, &ch.VulnCategorySlug, &ch.VulnerableCode,
		&ch.TargetVulnerability, &ch.ConceptualFix, &ch.VulnerableLines,
		&hintsJSON, &ch.Points, &ch.Status,
		&ch.ReviewerID, &ch.ReviewerNotes, &ch.ChallengeID,
		&ch.CreatedAt, &ch.UpdatedAt,
		&ch.AuthorUsername,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning community challenge: %w", err)
	}

	if err := json.Unmarshal(hintsJSON, &ch.Hints); err != nil {
		ch.Hints = []string{}
	}

	return &ch, nil
}

// scanMany scans multiple community challenges from rows.
func (r *CommunityRepository) scanMany(rows pgx.Rows) ([]model.CommunityChallenge, error) {
	items := make([]model.CommunityChallenge, 0)
	for rows.Next() {
		var ch model.CommunityChallenge
		var hintsJSON []byte

		err := rows.Scan(
			&ch.ID, &ch.AuthorID, &ch.Title, &ch.Description, &ch.Difficulty,
			&ch.LanguageSlug, &ch.VulnCategorySlug, &ch.VulnerableCode,
			&ch.TargetVulnerability, &ch.ConceptualFix, &ch.VulnerableLines,
			&hintsJSON, &ch.Points, &ch.Status,
			&ch.ReviewerID, &ch.ReviewerNotes, &ch.ChallengeID,
			&ch.CreatedAt, &ch.UpdatedAt,
			&ch.AuthorUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning community challenge row: %w", err)
		}

		if err := json.Unmarshal(hintsJSON, &ch.Hints); err != nil {
			ch.Hints = []string{}
		}

		items = append(items, ch)
	}
	return items, nil
}
