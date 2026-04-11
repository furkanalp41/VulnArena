package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type SubmissionRepository struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepository(pool *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{pool: pool}
}

func (r *SubmissionRepository) Create(ctx context.Context, sub *model.Submission) error {
	query := `
		INSERT INTO submissions (id, user_id, challenge_id, answer_text, target_lines, score, is_correct, is_first_blood, feedback, time_spent_sec)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		sub.ID, sub.UserID, sub.ChallengeID, sub.AnswerText, sub.TargetLines,
		sub.Score, sub.IsCorrect, sub.IsFirstBlood, sub.Feedback, sub.TimeSpentSec,
	).Scan(&sub.CreatedAt)
}

func (r *SubmissionRepository) ListByUserAndChallenge(ctx context.Context, userID, challengeID uuid.UUID) ([]model.Submission, error) {
	query := `
		SELECT id, user_id, challenge_id, answer_text, target_lines, score, is_correct, is_first_blood, feedback, time_spent_sec, created_at
		FROM submissions
		WHERE user_id = $1 AND challenge_id = $2
		ORDER BY created_at DESC
		LIMIT 20`

	rows, err := r.pool.Query(ctx, query, userID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("listing submissions: %w", err)
	}
	defer rows.Close()

	var subs []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ChallengeID, &s.AnswerText, &s.TargetLines,
			&s.Score, &s.IsCorrect, &s.IsFirstBlood, &s.Feedback, &s.TimeSpentSec, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning submission: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *SubmissionRepository) UpsertProgress(ctx context.Context, userID, challengeID uuid.UUID, score float64, passed bool) error {
	now := time.Now()
	status := model.ProgressAttempted
	if passed {
		status = model.ProgressSolved
	}

	query := `
		INSERT INTO user_challenge_progress (user_id, challenge_id, status, best_score, attempt_count, last_attempted, first_solved_at)
		VALUES ($1, $2, $3, $4, 1, $5, $6)
		ON CONFLICT (user_id, challenge_id) DO UPDATE SET
			status = CASE
				WHEN user_challenge_progress.status = 'solved' THEN 'solved'
				ELSE EXCLUDED.status
			END,
			best_score = GREATEST(user_challenge_progress.best_score, EXCLUDED.best_score),
			attempt_count = user_challenge_progress.attempt_count + 1,
			last_attempted = EXCLUDED.last_attempted,
			first_solved_at = CASE
				WHEN user_challenge_progress.first_solved_at IS NULL AND EXCLUDED.status = 'solved' THEN EXCLUDED.first_solved_at
				ELSE user_challenge_progress.first_solved_at
			END`

	var solvedAt *time.Time
	if passed {
		solvedAt = &now
	}

	_, err := r.pool.Exec(ctx, query, userID, challengeID, status, score, now, solvedAt)
	return err
}

func (r *SubmissionRepository) GetProgress(ctx context.Context, userID, challengeID uuid.UUID) (*model.UserChallengeProgress, error) {
	query := `
		SELECT user_id, challenge_id, status, best_score, attempt_count, first_solved_at, last_attempted
		FROM user_challenge_progress
		WHERE user_id = $1 AND challenge_id = $2`

	var p model.UserChallengeProgress
	err := r.pool.QueryRow(ctx, query, userID, challengeID).Scan(
		&p.UserID, &p.ChallengeID, &p.Status, &p.BestScore,
		&p.AttemptCount, &p.FirstSolvedAt, &p.LastAttempted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.UserChallengeProgress{
				UserID:      userID,
				ChallengeID: challengeID,
				Status:      model.ProgressNotStarted,
			}, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *SubmissionRepository) GetUserProgressMap(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]*model.UserChallengeProgress, error) {
	query := `
		SELECT user_id, challenge_id, status, best_score, attempt_count, first_solved_at, last_attempted
		FROM user_challenge_progress
		WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]*model.UserChallengeProgress)
	for rows.Next() {
		var p model.UserChallengeProgress
		if err := rows.Scan(
			&p.UserID, &p.ChallengeID, &p.Status, &p.BestScore,
			&p.AttemptCount, &p.FirstSolvedAt, &p.LastAttempted,
		); err != nil {
			return nil, err
		}
		result[p.ChallengeID] = &p
	}
	return result, nil
}

// ProgressSummary returns aggregated stats for a user.
type ProgressSummary struct {
	TotalSolved    int     `json:"total_solved"`
	TotalAttempted int     `json:"total_attempted"`
	TotalPoints    int     `json:"total_points"`
	AverageScore   float64 `json:"average_score"`
}

func (r *SubmissionRepository) GetProgressSummary(ctx context.Context, userID uuid.UUID) (*ProgressSummary, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE ucp.status = 'solved') AS total_solved,
			COUNT(*) FILTER (WHERE ucp.status IN ('attempted', 'solved')) AS total_attempted,
			COALESCE(SUM(c.points) FILTER (WHERE ucp.status = 'solved'), 0)
				+ COALESCE(SUM(CASE WHEN c.first_blood_user_id = ucp.user_id AND ucp.status = 'solved' THEN c.points / 4 ELSE 0 END), 0) AS total_points,
			COALESCE(AVG(ucp.best_score) FILTER (WHERE ucp.status IN ('attempted', 'solved')), 0) AS avg_score
		FROM user_challenge_progress ucp
		JOIN challenges c ON c.id = ucp.challenge_id
		WHERE ucp.user_id = $1`

	var s ProgressSummary
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.TotalSolved, &s.TotalAttempted, &s.TotalPoints, &s.AverageScore,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Ensure json import is used
var _ = json.RawMessage{}
