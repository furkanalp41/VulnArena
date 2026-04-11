package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type ChallengeRepository struct {
	pool *pgxpool.Pool
}

func NewChallengeRepository(pool *pgxpool.Pool) *ChallengeRepository {
	return &ChallengeRepository{pool: pool}
}

func (r *ChallengeRepository) List(ctx context.Context, filter model.ChallengeFilter) ([]model.ChallengeListItem, int, error) {
	where := []string{"c.is_published = TRUE"}
	args := []any{}
	argIdx := 1

	if filter.LanguageSlug != "" {
		where = append(where, fmt.Sprintf("l.slug = $%d", argIdx))
		args = append(args, filter.LanguageSlug)
		argIdx++
	}
	if filter.VulnCategorySlug != "" {
		where = append(where, fmt.Sprintf("v.slug = $%d", argIdx))
		args = append(args, filter.VulnCategorySlug)
		argIdx++
	}
	if filter.DifficultyMin > 0 {
		where = append(where, fmt.Sprintf("c.difficulty >= $%d", argIdx))
		args = append(args, filter.DifficultyMin)
		argIdx++
	}
	if filter.DifficultyMax > 0 {
		where = append(where, fmt.Sprintf("c.difficulty <= $%d", argIdx))
		args = append(args, filter.DifficultyMax)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM challenges c
		JOIN languages l ON c.language_id = l.id
		JOIN vuln_categories v ON c.vuln_category_id = v.id
		WHERE %s`, whereClause)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting challenges: %w", err)
	}

	// List query
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	listQuery := fmt.Sprintf(`
		SELECT c.id, c.title, c.slug, c.description, c.difficulty, c.points, c.line_count,
			l.id, l.slug, l.name,
			v.id, v.slug, v.name, v.owasp_ref
		FROM challenges c
		JOIN languages l ON c.language_id = l.id
		JOIN vuln_categories v ON c.vuln_category_id = v.id
		WHERE %s
		ORDER BY c.difficulty ASC, c.created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing challenges: %w", err)
	}
	defer rows.Close()

	var items []model.ChallengeListItem
	for rows.Next() {
		var item model.ChallengeListItem
		var lang model.Language
		var cat model.VulnCategory
		var owaspRef *string

		if err := rows.Scan(
			&item.ID, &item.Title, &item.Slug, &item.Description,
			&item.Difficulty, &item.Points, &item.LineCount,
			&lang.ID, &lang.Slug, &lang.Name,
			&cat.ID, &cat.Slug, &cat.Name, &owaspRef,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning challenge: %w", err)
		}

		if owaspRef != nil {
			cat.OWASPRef = *owaspRef
		}
		item.Language = &lang
		item.VulnCategory = &cat
		items = append(items, item)
	}

	return items, total, nil
}

func (r *ChallengeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Challenge, error) {
	query := `
		SELECT c.id, c.title, c.slug, c.description, c.difficulty,
			c.language_id, c.vuln_category_id,
			c.vulnerable_code, c.target_vulnerability, c.conceptual_fix,
			c.vulnerable_lines, c.cve_reference,
			c.hints, c.points, c.line_count, c.is_published,
			c.first_blood_user_id,
			c.created_at, c.updated_at,
			l.id, l.slug, l.name,
			v.id, v.slug, v.name, v.description, v.owasp_ref
		FROM challenges c
		JOIN languages l ON c.language_id = l.id
		JOIN vuln_categories v ON c.vuln_category_id = v.id
		WHERE c.id = $1`

	var ch model.Challenge
	var lang model.Language
	var cat model.VulnCategory
	var catDesc, owaspRef *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ch.ID, &ch.Title, &ch.Slug, &ch.Description, &ch.Difficulty,
		&ch.LanguageID, &ch.VulnCategoryID,
		&ch.VulnerableCode, &ch.TargetVulnerability, &ch.ConceptualFix,
		&ch.VulnerableLines, &ch.CVEReference,
		&ch.Hints, &ch.Points, &ch.LineCount, &ch.IsPublished,
		&ch.FirstBloodUserID,
		&ch.CreatedAt, &ch.UpdatedAt,
		&lang.ID, &lang.Slug, &lang.Name,
		&cat.ID, &cat.Slug, &cat.Name, &catDesc, &owaspRef,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting challenge: %w", err)
	}

	if catDesc != nil {
		cat.Description = *catDesc
	}
	if owaspRef != nil {
		cat.OWASPRef = *owaspRef
	}
	ch.Language = &lang
	ch.VulnCategory = &cat

	return &ch, nil
}

func (r *ChallengeRepository) Insert(ctx context.Context, ch *model.Challenge) error {
	query := `
		INSERT INTO challenges (id, title, slug, description, difficulty, language_id, vuln_category_id,
			vulnerable_code, target_vulnerability, conceptual_fix, vulnerable_lines, cve_reference,
			hints, points, line_count, is_published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (slug) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			difficulty = EXCLUDED.difficulty,
			vulnerable_code = EXCLUDED.vulnerable_code,
			target_vulnerability = EXCLUDED.target_vulnerability,
			conceptual_fix = EXCLUDED.conceptual_fix,
			vulnerable_lines = EXCLUDED.vulnerable_lines,
			cve_reference = EXCLUDED.cve_reference,
			hints = EXCLUDED.hints,
			points = EXCLUDED.points,
			line_count = EXCLUDED.line_count,
			updated_at = NOW()
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		ch.ID, ch.Title, ch.Slug, ch.Description, ch.Difficulty,
		ch.LanguageID, ch.VulnCategoryID,
		ch.VulnerableCode, ch.TargetVulnerability, ch.ConceptualFix,
		ch.VulnerableLines, ch.CVEReference,
		ch.Hints, ch.Points, ch.LineCount, ch.IsPublished,
	).Scan(&ch.CreatedAt, &ch.UpdatedAt)
}

// ClaimFirstBlood atomically sets first_blood_user_id if not already claimed.
// Returns true if this user claimed first blood, false if already taken.
func (r *ChallengeRepository) ClaimFirstBlood(ctx context.Context, challengeID, userID uuid.UUID) (bool, error) {
	query := `
		UPDATE challenges
		SET first_blood_user_id = $2
		WHERE id = $1 AND first_blood_user_id IS NULL`

	tag, err := r.pool.Exec(ctx, query, challengeID, userID)
	if err != nil {
		return false, fmt.Errorf("claiming first blood: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// Update modifies an existing challenge (admin operation).
func (r *ChallengeRepository) Update(ctx context.Context, ch *model.Challenge) error {
	query := `
		UPDATE challenges SET
			title = $2, description = $3, difficulty = $4,
			language_id = $5, vuln_category_id = $6,
			vulnerable_code = $7, target_vulnerability = $8, conceptual_fix = $9,
			vulnerable_lines = $10, cve_reference = $11,
			hints = $12, points = $13, line_count = $14, is_published = $15,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		ch.ID, ch.Title, ch.Description, ch.Difficulty,
		ch.LanguageID, ch.VulnCategoryID,
		ch.VulnerableCode, ch.TargetVulnerability, ch.ConceptualFix,
		ch.VulnerableLines, ch.CVEReference,
		ch.Hints, ch.Points, ch.LineCount, ch.IsPublished,
	).Scan(&ch.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("updating challenge: %w", err)
	}
	return nil
}

func (r *ChallengeRepository) GetLanguageBySlug(ctx context.Context, slug string) (*model.Language, error) {
	var lang model.Language
	err := r.pool.QueryRow(ctx, `SELECT id, slug, name FROM languages WHERE slug = $1`, slug).
		Scan(&lang.ID, &lang.Slug, &lang.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &lang, nil
}

func (r *ChallengeRepository) GetVulnCategoryBySlug(ctx context.Context, slug string) (*model.VulnCategory, error) {
	var cat model.VulnCategory
	var desc, owasp *string
	err := r.pool.QueryRow(ctx, `SELECT id, slug, name, description, owasp_ref FROM vuln_categories WHERE slug = $1`, slug).
		Scan(&cat.ID, &cat.Slug, &cat.Name, &desc, &owasp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if desc != nil {
		cat.Description = *desc
	}
	if owasp != nil {
		cat.OWASPRef = *owasp
	}
	return &cat, nil
}
