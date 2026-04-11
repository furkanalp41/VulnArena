package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type TelemetryRepository struct {
	pool *pgxpool.Pool
}

func NewTelemetryRepository(pool *pgxpool.Pool) *TelemetryRepository {
	return &TelemetryRepository{pool: pool}
}

// GetOverallStats returns aggregated statistics for a user.
func (r *TelemetryRepository) GetOverallStats(ctx context.Context, userID uuid.UUID) (*model.OverallStats, error) {
	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM user_challenge_progress WHERE user_id = $1 AND status = 'solved'), 0),
			COALESCE((SELECT COUNT(*) FROM challenges WHERE is_published = TRUE), 0),
			COALESCE((SELECT COUNT(*) FROM user_challenge_progress WHERE user_id = $1 AND status IN ('attempted','solved')), 0),
			COALESCE((
				SELECT SUM(c.points) + COALESCE(SUM(CASE WHEN c.first_blood_user_id = $1 THEN c.points / 4 ELSE 0 END), 0)
				FROM user_challenge_progress ucp
				JOIN challenges c ON c.id = ucp.challenge_id
				WHERE ucp.user_id = $1 AND ucp.status = 'solved'
			), 0),
			COALESCE((SELECT AVG(ucp.best_score) FROM user_challenge_progress ucp WHERE ucp.user_id = $1 AND ucp.status IN ('attempted','solved')), 0),
			COALESCE((SELECT COUNT(*) FROM user_lesson_progress WHERE user_id = $1 AND completed = TRUE), 0)`

	var s model.OverallStats
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.TotalSolved, &s.TotalAvailable, &s.TotalAttempted,
		&s.TotalPoints, &s.AverageScore, &s.LessonsRead,
	)
	if err != nil {
		return nil, fmt.Errorf("getting overall stats: %w", err)
	}

	// Calculate streak (consecutive days with activity, counting back from today)
	s.CurrentStreak, _ = r.calculateStreak(ctx, userID)

	return &s, nil
}

func (r *TelemetryRepository) calculateStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	// Get distinct dates with submissions, ordered descending
	query := `
		SELECT DISTINCT DATE(created_at) AS activity_date
		FROM submissions
		WHERE user_id = $1
		ORDER BY activity_date DESC
		LIMIT 365`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	streak := 0
	today := time.Now().Truncate(24 * time.Hour)
	expected := today

	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return streak, err
		}
		d = d.Truncate(24 * time.Hour)

		// Allow today or yesterday as start
		if streak == 0 {
			diff := today.Sub(d).Hours() / 24
			if diff > 1 {
				return 0, nil // No recent activity
			}
			expected = d
		}

		if d.Equal(expected) {
			streak++
			expected = expected.AddDate(0, 0, -1)
		} else if d.Before(expected) {
			break
		}
	}

	return streak, nil
}

// GetSkillRadar returns per-category proficiency scores.
func (r *TelemetryRepository) GetSkillRadar(ctx context.Context, userID uuid.UUID) ([]model.SkillRadarPoint, error) {
	query := `
		SELECT
			vc.name,
			vc.slug,
			COUNT(c.id) AS total_challenges,
			COUNT(ucp.challenge_id) FILTER (WHERE ucp.status = 'solved') AS solved,
			COALESCE(AVG(ucp.best_score) FILTER (WHERE ucp.status IN ('attempted','solved')), 0) AS avg_score
		FROM vuln_categories vc
		LEFT JOIN challenges c ON c.vuln_category_id = vc.id AND c.is_published = TRUE
		LEFT JOIN user_challenge_progress ucp ON ucp.challenge_id = c.id AND ucp.user_id = $1
		GROUP BY vc.id, vc.name, vc.slug
		HAVING COUNT(c.id) > 0
		ORDER BY vc.name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("getting skill radar: %w", err)
	}
	defer rows.Close()

	points := make([]model.SkillRadarPoint, 0)
	for rows.Next() {
		var p model.SkillRadarPoint
		if err := rows.Scan(&p.Category, &p.Slug, &p.Total, &p.Solved, &p.Score); err != nil {
			return nil, fmt.Errorf("scanning skill point: %w", err)
		}
		// Blend completion ratio and score for a composite proficiency
		if p.Total > 0 {
			completionRatio := float64(p.Solved) / float64(p.Total) * 100
			p.Score = (p.Score*0.6 + completionRatio*0.4)
		}
		points = append(points, p)
	}

	return points, nil
}

// GetRecentActivity returns the latest activity entries for a user.
func (r *TelemetryRepository) GetRecentActivity(ctx context.Context, userID uuid.UUID, limit int) ([]model.ActivityEntry, error) {
	if limit <= 0 {
		limit = 15
	}

	query := `
		(
			SELECT
				CASE WHEN s.is_correct THEN 'challenge_solved' ELSE 'challenge_attempted' END AS type,
				c.title,
				CASE WHEN s.is_correct THEN c.points ELSE 0 END AS points,
				s.score,
				s.created_at AS occurred_at
			FROM submissions s
			JOIN challenges c ON c.id = s.challenge_id
			WHERE s.user_id = $1
			ORDER BY s.created_at DESC
			LIMIT $2
		)
		UNION ALL
		(
			SELECT
				'lesson_completed' AS type,
				l.title,
				0 AS points,
				0 AS score,
				ulp.completed_at AS occurred_at
			FROM user_lesson_progress ulp
			JOIN lessons l ON l.id = ulp.lesson_id
			WHERE ulp.user_id = $1 AND ulp.completed = TRUE AND ulp.completed_at IS NOT NULL
			ORDER BY ulp.completed_at DESC
			LIMIT $2
		)
		ORDER BY occurred_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("getting recent activity: %w", err)
	}
	defer rows.Close()

	entries := make([]model.ActivityEntry, 0)
	for rows.Next() {
		var e model.ActivityEntry
		if err := rows.Scan(&e.Type, &e.Title, &e.Points, &e.Score, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("scanning activity: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// LeaderboardRow holds raw leaderboard data from the database.
type LeaderboardRow struct {
	Username    string
	DisplayName string
	AvatarURL   string
	TotalXP     int
	TotalSolved int
}

// GetLeaderboard returns the top users ranked by total XP.
func (r *TelemetryRepository) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardRow, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_url, ''),
			(COALESCE(SUM(c.points), 0) + COALESCE(SUM(CASE WHEN c.first_blood_user_id = u.id THEN c.points / 4 ELSE 0 END), 0))::int AS total_xp,
			COUNT(ucp.challenge_id)::int AS total_solved
		FROM users u
		LEFT JOIN user_challenge_progress ucp ON ucp.user_id = u.id AND ucp.status = 'solved'
		LEFT JOIN challenges c ON c.id = ucp.challenge_id
		GROUP BY u.id, u.username, u.display_name, u.avatar_url
		HAVING COALESCE(SUM(c.points), 0) > 0
		ORDER BY total_xp DESC, total_solved DESC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting leaderboard: %w", err)
	}
	defer rows.Close()

	results := make([]LeaderboardRow, 0)
	for rows.Next() {
		var row LeaderboardRow
		if err := rows.Scan(&row.Username, &row.DisplayName, &row.AvatarURL, &row.TotalXP, &row.TotalSolved); err != nil {
			return nil, fmt.Errorf("scanning leaderboard row: %w", err)
		}
		results = append(results, row)
	}

	return results, nil
}

// GetPublicActivity returns only solved challenge entries for a user (pwned labs).
func (r *TelemetryRepository) GetPublicActivity(ctx context.Context, userID uuid.UUID, limit int) ([]model.ActivityEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			'challenge_solved' AS type,
			c.title,
			c.points,
			s.score,
			s.created_at AS occurred_at
		FROM submissions s
		JOIN challenges c ON c.id = s.challenge_id
		WHERE s.user_id = $1 AND s.is_correct = TRUE
		ORDER BY s.created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("getting public activity: %w", err)
	}
	defer rows.Close()

	entries := make([]model.ActivityEntry, 0)
	for rows.Next() {
		var e model.ActivityEntry
		if err := rows.Scan(&e.Type, &e.Title, &e.Points, &e.Score, &e.OccurredAt); err != nil {
			return nil, fmt.Errorf("scanning public activity: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// GetNextChallenge suggests the best next challenge for a user based on
// their skill level: the easiest unsolved published challenge.
func (r *TelemetryRepository) GetNextChallenge(ctx context.Context, userID uuid.UUID) (*model.ChallengeListItem, error) {
	query := `
		SELECT c.id, c.title, c.slug, c.description, c.difficulty, c.points, c.line_count,
			l.id, l.slug, l.name,
			v.id, v.slug, v.name, v.owasp_ref
		FROM challenges c
		JOIN languages l ON c.language_id = l.id
		JOIN vuln_categories v ON c.vuln_category_id = v.id
		LEFT JOIN user_challenge_progress ucp ON ucp.challenge_id = c.id AND ucp.user_id = $1
		WHERE c.is_published = TRUE
			AND (ucp.status IS NULL OR ucp.status != 'solved')
		ORDER BY c.difficulty ASC, c.created_at ASC
		LIMIT 1`

	var item model.ChallengeListItem
	var lang model.Language
	var cat model.VulnCategory
	var owaspRef *string

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&item.ID, &item.Title, &item.Slug, &item.Description,
		&item.Difficulty, &item.Points, &item.LineCount,
		&lang.ID, &lang.Slug, &lang.Name,
		&cat.ID, &cat.Slug, &cat.Name, &owaspRef,
	)
	if err != nil {
		return nil, nil // No unsolved challenges — return nil, not error
	}

	if owaspRef != nil {
		cat.OWASPRef = *owaspRef
	}
	item.Language = &lang
	item.VulnCategory = &cat

	return &item, nil
}
