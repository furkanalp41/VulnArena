package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type TeamService struct {
	teamRepo *repository.TeamRepository
	logger   *slog.Logger
}

func NewTeamService(teamRepo *repository.TeamRepository, logger *slog.Logger) *TeamService {
	return &TeamService{teamRepo: teamRepo, logger: logger}
}

type CreateTeamInput struct {
	Name        string `json:"name"`
	Tag         string `json:"tag"`
	Description string `json:"description"`
}

var tagRegex = regexp.MustCompile(`^[A-Z0-9]{2,4}$`)

// CreateTeam creates a new team and adds the creator as leader.
func (s *TeamService) CreateTeam(ctx context.Context, userID uuid.UUID, input CreateTeamInput) (*model.TeamWithMembers, error) {
	// Validate tag
	tag := strings.ToUpper(strings.TrimSpace(input.Tag))
	if !tagRegex.MatchString(tag) {
		return nil, fmt.Errorf("tag must be 2-4 uppercase alphanumeric characters")
	}

	name := strings.TrimSpace(input.Name)
	if len(name) < 2 || len(name) > 100 {
		return nil, fmt.Errorf("team name must be 2-100 characters")
	}

	// Check if user is already in a team
	existing, err := s.teamRepo.GetUserTeam(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checking existing team: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("you are already in a team")
	}

	team := &model.Team{
		ID:          uuid.New(),
		Name:        name,
		Tag:         tag,
		Description: strings.TrimSpace(input.Description),
		CreatedBy:   userID,
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("creating team: %w", err)
	}

	// Add creator as leader
	if err := s.teamRepo.AddMember(ctx, team.ID, userID, "leader"); err != nil {
		return nil, fmt.Errorf("adding leader: %w", err)
	}

	s.logger.Info("team created", slog.String("tag", tag), slog.String("name", name))

	return s.GetTeam(ctx, tag)
}

// GetTeam returns a team with its members and computed stats.
func (s *TeamService) GetTeam(ctx context.Context, tag string) (*model.TeamWithMembers, error) {
	team, err := s.teamRepo.GetByTag(ctx, strings.ToUpper(tag))
	if err != nil {
		return nil, fmt.Errorf("getting team: %w", err)
	}
	if team == nil {
		return nil, nil
	}

	members, err := s.teamRepo.GetMembers(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("getting members: %w", err)
	}

	totalXP, totalSolved, err := s.teamRepo.GetTeamXPAndSolved(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("getting team stats: %w", err)
	}

	return &model.TeamWithMembers{
		Team:        *team,
		Members:     members,
		TotalXP:     totalXP,
		TotalSolved: totalSolved,
	}, nil
}

// ListTeams returns all teams.
func (s *TeamService) ListTeams(ctx context.Context) ([]model.Team, error) {
	return s.teamRepo.List(ctx)
}

// JoinTeam adds a user to an existing team as a member.
func (s *TeamService) JoinTeam(ctx context.Context, userID uuid.UUID, tag string) error {
	// Check if user is already in a team
	existing, err := s.teamRepo.GetUserTeam(ctx, userID)
	if err != nil {
		return fmt.Errorf("checking existing team: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("you are already in a team")
	}

	team, err := s.teamRepo.GetByTag(ctx, strings.ToUpper(tag))
	if err != nil {
		return fmt.Errorf("finding team: %w", err)
	}
	if team == nil {
		return fmt.Errorf("team not found")
	}

	if err := s.teamRepo.AddMember(ctx, team.ID, userID, "member"); err != nil {
		return fmt.Errorf("joining team: %w", err)
	}

	s.logger.Info("user joined team", slog.String("tag", tag))
	return nil
}

// LeaveTeam removes a user from their team. If the leader leaves, promote the oldest member or delete the team.
func (s *TeamService) LeaveTeam(ctx context.Context, userID uuid.UUID) error {
	team, err := s.teamRepo.GetUserTeam(ctx, userID)
	if err != nil {
		return fmt.Errorf("finding user team: %w", err)
	}
	if team == nil {
		return fmt.Errorf("you are not in a team")
	}

	// Remove the member first
	if err := s.teamRepo.RemoveMember(ctx, team.ID, userID); err != nil {
		return fmt.Errorf("leaving team: %w", err)
	}

	// Check if this was the leader
	if team.CreatedBy == userID {
		memberCount, err := s.teamRepo.GetMemberCount(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("checking remaining members: %w", err)
		}

		if memberCount == 0 {
			// No members left — delete the team
			if err := s.teamRepo.DeleteTeam(ctx, team.ID); err != nil {
				return fmt.Errorf("deleting empty team: %w", err)
			}
			s.logger.Info("team dissolved", slog.String("tag", team.Tag))
		} else {
			// Promote the oldest remaining member
			if err := s.teamRepo.PromoteOldestMember(ctx, team.ID); err != nil {
				return fmt.Errorf("promoting new leader: %w", err)
			}
			s.logger.Info("new leader promoted", slog.String("tag", team.Tag))
		}
	}

	return nil
}

// GetTeamLeaderboard returns the top teams by combined XP.
func (s *TeamService) GetTeamLeaderboard(ctx context.Context) ([]model.TeamLeaderboardEntry, error) {
	return s.teamRepo.GetTeamLeaderboard(ctx, 50)
}

// GetUserTeam returns the team a user belongs to, or nil.
func (s *TeamService) GetUserTeam(ctx context.Context, userID uuid.UUID) (*model.Team, error) {
	return s.teamRepo.GetUserTeam(ctx, userID)
}
