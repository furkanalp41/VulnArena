package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type AchievementRepository struct {
	pool *pgxpool.Pool
}

func NewAchievementRepository(pool *pgxpool.Pool) *AchievementRepository {
	return &AchievementRepository{pool: pool}
}

// Insert creates or updates an achievement by slug (upsert).
func (r *AchievementRepository) Insert(ctx context.Context, a *model.Achievement) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO achievements (id, slug, name, description, icon_svg, category, xp_reward)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (slug) DO UPDATE SET
		   name = EXCLUDED.name,
		   description = EXCLUDED.description,
		   icon_svg = EXCLUDED.icon_svg,
		   category = EXCLUDED.category,
		   xp_reward = EXCLUDED.xp_reward`,
		a.ID, a.Slug, a.Name, a.Description, a.IconSVG, a.Category, a.XPReward)
	if err != nil {
		return fmt.Errorf("inserting achievement %q: %w", a.Slug, err)
	}
	return nil
}

// ListAll returns every achievement in the catalog.
func (r *AchievementRepository) ListAll(ctx context.Context) ([]model.Achievement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, slug, name, description, icon_svg, category, xp_reward, created_at
		 FROM achievements ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing achievements: %w", err)
	}
	defer rows.Close()

	achievements := make([]model.Achievement, 0)
	for rows.Next() {
		var a model.Achievement
		if err := rows.Scan(&a.ID, &a.Slug, &a.Name, &a.Description, &a.IconSVG, &a.Category, &a.XPReward, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning achievement: %w", err)
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

// GetUserAchievements returns all badges a user has unlocked, with full achievement data.
func (r *AchievementRepository) GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]model.UserAchievement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT a.id, a.slug, a.name, a.description, a.icon_svg, a.category, a.xp_reward, a.created_at, ua.unlocked_at
		 FROM user_achievements ua
		 JOIN achievements a ON a.id = ua.achievement_id
		 WHERE ua.user_id = $1
		 ORDER BY ua.unlocked_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user achievements: %w", err)
	}
	defer rows.Close()

	result := make([]model.UserAchievement, 0)
	for rows.Next() {
		var ua model.UserAchievement
		if err := rows.Scan(
			&ua.Achievement.ID, &ua.Achievement.Slug, &ua.Achievement.Name,
			&ua.Achievement.Description, &ua.Achievement.IconSVG, &ua.Achievement.Category,
			&ua.Achievement.XPReward, &ua.Achievement.CreatedAt, &ua.UnlockedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning user achievement: %w", err)
		}
		result = append(result, ua)
	}
	return result, nil
}

// HasAchievement checks if a user already holds a specific badge.
func (r *AchievementRepository) HasAchievement(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_achievements ua
			JOIN achievements a ON a.id = ua.achievement_id
			WHERE ua.user_id = $1 AND a.slug = $2
		)`, userID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking achievement %q: %w", slug, err)
	}
	return exists, nil
}

// Grant awards a badge to a user. Idempotent — does nothing if already held.
// Returns the achievement and whether it was newly granted.
func (r *AchievementRepository) Grant(ctx context.Context, userID uuid.UUID, slug string) (*model.UserAchievement, bool, error) {
	// Look up the achievement by slug
	var a model.Achievement
	err := r.pool.QueryRow(ctx,
		`SELECT id, slug, name, description, icon_svg, category, xp_reward, created_at
		 FROM achievements WHERE slug = $1`, slug).Scan(
		&a.ID, &a.Slug, &a.Name, &a.Description, &a.IconSVG, &a.Category, &a.XPReward, &a.CreatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("achievement %q not found: %w", slug, err)
	}

	// Attempt to grant (idempotent via ON CONFLICT DO NOTHING)
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO user_achievements (user_id, achievement_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, achievement_id) DO NOTHING`, userID, a.ID)
	if err != nil {
		return nil, false, fmt.Errorf("granting achievement: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Already held
		return nil, false, nil
	}

	// Newly granted — read back the unlock timestamp
	var result model.UserAchievement
	result.Achievement = a
	err = r.pool.QueryRow(ctx,
		`SELECT unlocked_at FROM user_achievements WHERE user_id = $1 AND achievement_id = $2`,
		userID, a.ID).Scan(&result.UnlockedAt)
	if err != nil {
		return nil, false, fmt.Errorf("reading unlock time: %w", err)
	}

	return &result, true, nil
}

// GetSolvedCountByCategory returns how many challenges a user has solved in a specific vuln category.
func (r *AchievementRepository) GetSolvedCountByCategory(ctx context.Context, userID uuid.UUID, categorySlug string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_challenge_progress ucp
		 JOIN challenges c ON c.id = ucp.challenge_id
		 JOIN vuln_categories vc ON vc.id = c.vuln_category_id
		 WHERE ucp.user_id = $1 AND ucp.status = 'solved' AND vc.slug = $2`,
		userID, categorySlug).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting solved by category: %w", err)
	}
	return count, nil
}

// GetTotalSolved returns the total number of challenges a user has solved.
func (r *AchievementRepository) GetTotalSolved(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_challenge_progress
		 WHERE user_id = $1 AND status = 'solved'`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting total solved: %w", err)
	}
	return count, nil
}

// GetCurrentStreak returns the user's consecutive-day solving streak.
func (r *AchievementRepository) GetCurrentStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	// Use a recursive CTE to count consecutive days with correct submissions
	var streak int
	err := r.pool.QueryRow(ctx,
		`WITH solve_dates AS (
			SELECT DISTINCT DATE(created_at) AS d
			FROM submissions
			WHERE user_id = $1 AND is_correct = TRUE
		),
		streak AS (
			SELECT d, 1 AS len FROM solve_dates WHERE d >= CURRENT_DATE - INTERVAL '1 day'
			UNION ALL
			SELECT sd.d, s.len + 1
			FROM solve_dates sd
			JOIN streak s ON sd.d = s.d - INTERVAL '1 day'
		)
		SELECT COALESCE(MAX(len), 0) FROM streak`, userID).Scan(&streak)
	if err != nil {
		return 0, fmt.Errorf("calculating streak: %w", err)
	}
	return streak, nil
}
