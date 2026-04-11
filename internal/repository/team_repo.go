package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
)

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// Create inserts a new team.
func (r *TeamRepository) Create(ctx context.Context, t *model.Team) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO teams (id, name, tag, description, avatar_url, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		t.ID, t.Name, t.Tag, t.Description, t.AvatarURL, t.CreatedBy,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

// GetByID returns a team by its UUID.
func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, tag, description, avatar_url, created_by, created_at, updated_at
		 FROM teams WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Tag, &t.Description, &t.AvatarURL, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting team by id: %w", err)
	}
	return &t, nil
}

// GetByTag returns a team by its unique tag.
func (r *TeamRepository) GetByTag(ctx context.Context, tag string) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, tag, description, avatar_url, created_by, created_at, updated_at
		 FROM teams WHERE tag = $1`, tag,
	).Scan(&t.ID, &t.Name, &t.Tag, &t.Description, &t.AvatarURL, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting team by tag: %w", err)
	}
	return &t, nil
}

// List returns all teams ordered by creation date.
func (r *TeamRepository) List(ctx context.Context) ([]model.Team, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, tag, description, avatar_url, created_by, created_at, updated_at
		 FROM teams ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	defer rows.Close()

	teams := make([]model.Team, 0)
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Tag, &t.Description, &t.AvatarURL, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning team: %w", err)
		}
		teams = append(teams, t)
	}
	return teams, nil
}

// AddMember inserts a team member. Fails if the user is already in any team (unique index).
func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role)
		 VALUES ($1, $2, $3)`, teamID, userID, role)
	if err != nil {
		return fmt.Errorf("adding team member: %w", err)
	}
	return nil
}

// RemoveMember removes a user from a team.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	if err != nil {
		return fmt.Errorf("removing team member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user is not a member of this team")
	}
	return nil
}

// GetMembers returns all members of a team, joined with user data.
func (r *TeamRepository) GetMembers(ctx context.Context, teamID uuid.UUID) ([]model.TeamMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tm.team_id, tm.user_id, tm.role, tm.joined_at, u.username, COALESCE(u.display_name, '')
		 FROM team_members tm
		 JOIN users u ON u.id = tm.user_id
		 WHERE tm.team_id = $1
		 ORDER BY tm.joined_at ASC`, teamID)
	if err != nil {
		return nil, fmt.Errorf("getting team members: %w", err)
	}
	defer rows.Close()

	members := make([]model.TeamMember, 0)
	for rows.Next() {
		var m model.TeamMember
		if err := rows.Scan(&m.TeamID, &m.UserID, &m.Role, &m.JoinedAt, &m.Username, &m.DisplayName); err != nil {
			return nil, fmt.Errorf("scanning team member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

// GetUserTeam returns the team a user belongs to, or nil if they're not in any team.
func (r *TeamRepository) GetUserTeam(ctx context.Context, userID uuid.UUID) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`SELECT t.id, t.name, t.tag, t.description, t.avatar_url, t.created_by, t.created_at, t.updated_at
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = $1`, userID,
	).Scan(&t.ID, &t.Name, &t.Tag, &t.Description, &t.AvatarURL, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user team: %w", err)
	}
	return &t, nil
}

// GetMemberCount returns the number of members in a team.
func (r *TeamRepository) GetMemberCount(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM team_members WHERE team_id = $1`, teamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting team members: %w", err)
	}
	return count, nil
}

// DeleteTeam removes a team entirely.
func (r *TeamRepository) DeleteTeam(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, teamID)
	if err != nil {
		return fmt.Errorf("deleting team: %w", err)
	}
	return nil
}

// PromoteOldestMember promotes the oldest non-leader member to leader.
func (r *TeamRepository) PromoteOldestMember(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE team_members SET role = 'leader'
		 WHERE team_id = $1 AND user_id = (
		   SELECT user_id FROM team_members
		   WHERE team_id = $1 AND role != 'leader'
		   ORDER BY joined_at ASC LIMIT 1
		 )`, teamID)
	if err != nil {
		return fmt.Errorf("promoting oldest member: %w", err)
	}
	return nil
}

// GetTeamLeaderboard returns teams ranked by combined member XP.
func (r *TeamRepository) GetTeamLeaderboard(ctx context.Context, limit int) ([]model.TeamLeaderboardEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.name, t.tag,
		        COUNT(DISTINCT tm.user_id) AS member_count,
		        COALESCE(SUM(xp.total_xp), 0) AS total_xp,
		        COALESCE(SUM(xp.total_solved), 0) AS total_solved
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 LEFT JOIN (
		   SELECT ucp.user_id,
		          SUM(c.points) AS total_xp,
		          COUNT(*) AS total_solved
		   FROM user_challenge_progress ucp
		   JOIN challenges c ON c.id = ucp.challenge_id
		   WHERE ucp.status = 'solved'
		   GROUP BY ucp.user_id
		 ) xp ON xp.user_id = tm.user_id
		 GROUP BY t.id, t.name, t.tag
		 ORDER BY total_xp DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("getting team leaderboard: %w", err)
	}
	defer rows.Close()

	entries := make([]model.TeamLeaderboardEntry, 0)
	rank := 0
	for rows.Next() {
		rank++
		var e model.TeamLeaderboardEntry
		if err := rows.Scan(&e.TeamName, &e.Tag, &e.MemberCount, &e.TotalXP, &e.TotalSolved); err != nil {
			return nil, fmt.Errorf("scanning leaderboard entry: %w", err)
		}
		e.Rank = rank
		entries = append(entries, e)
	}
	return entries, nil
}

// GetTeamXPAndSolved returns the combined XP and solved count for a team.
func (r *TeamRepository) GetTeamXPAndSolved(ctx context.Context, teamID uuid.UUID) (totalXP int, totalSolved int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(xp.total_xp), 0), COALESCE(SUM(xp.total_solved), 0)
		 FROM team_members tm
		 LEFT JOIN (
		   SELECT ucp.user_id,
		          SUM(c.points) AS total_xp,
		          COUNT(*) AS total_solved
		   FROM user_challenge_progress ucp
		   JOIN challenges c ON c.id = ucp.challenge_id
		   WHERE ucp.status = 'solved'
		   GROUP BY ucp.user_id
		 ) xp ON xp.user_id = tm.user_id
		 WHERE tm.team_id = $1`, teamID).Scan(&totalXP, &totalSolved)
	if err != nil {
		return 0, 0, fmt.Errorf("getting team XP: %w", err)
	}
	return totalXP, totalSolved, nil
}
