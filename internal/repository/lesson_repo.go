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

type LessonRepository struct {
	pool *pgxpool.Pool
}

func NewLessonRepository(pool *pgxpool.Pool) *LessonRepository {
	return &LessonRepository{pool: pool}
}

func (r *LessonRepository) List(ctx context.Context, filter model.LessonFilter) ([]model.LessonListItem, int, error) {
	whereClause := "is_published = TRUE"
	args := []any{}
	argIdx := 1

	if filter.Category != "" {
		whereClause += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, filter.Category)
		argIdx++
	}

	// Count
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM lessons WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting lessons: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	listQ := fmt.Sprintf(`
		SELECT id, title, slug, category, description, difficulty, read_time_min, tags
		FROM lessons
		WHERE %s
		ORDER BY difficulty ASC, created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing lessons: %w", err)
	}
	defer rows.Close()

	var items []model.LessonListItem
	for rows.Next() {
		var item model.LessonListItem
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Slug, &item.Category,
			&item.Description, &item.Difficulty, &item.ReadTimeMin, &item.Tags,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning lesson: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (r *LessonRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Lesson, error) {
	query := `
		SELECT id, title, slug, category, description, content, difficulty,
			read_time_min, tags, is_published, created_at, updated_at
		FROM lessons
		WHERE id = $1`

	var l model.Lesson
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.Title, &l.Slug, &l.Category, &l.Description,
		&l.Content, &l.Difficulty, &l.ReadTimeMin, &l.Tags,
		&l.IsPublished, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting lesson: %w", err)
	}
	return &l, nil
}

func (r *LessonRepository) Insert(ctx context.Context, l *model.Lesson) error {
	query := `
		INSERT INTO lessons (id, title, slug, category, description, content, difficulty, read_time_min, tags, is_published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (slug) DO UPDATE SET
			title = EXCLUDED.title,
			category = EXCLUDED.category,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			difficulty = EXCLUDED.difficulty,
			read_time_min = EXCLUDED.read_time_min,
			tags = EXCLUDED.tags,
			updated_at = NOW()
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		l.ID, l.Title, l.Slug, l.Category, l.Description,
		l.Content, l.Difficulty, l.ReadTimeMin, l.Tags, l.IsPublished,
	).Scan(&l.CreatedAt, &l.UpdatedAt)
}
