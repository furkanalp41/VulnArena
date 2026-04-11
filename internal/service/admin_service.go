package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type AdminService struct {
	pool          *pgxpool.Pool
	challengeRepo *repository.ChallengeRepository
	lessonRepo    *repository.LessonRepository
	communityRepo *repository.CommunityRepository
}

func NewAdminService(
	pool *pgxpool.Pool,
	challengeRepo *repository.ChallengeRepository,
	lessonRepo *repository.LessonRepository,
	communityRepo *repository.CommunityRepository,
) *AdminService {
	return &AdminService{
		pool:          pool,
		challengeRepo: challengeRepo,
		lessonRepo:    lessonRepo,
		communityRepo: communityRepo,
	}
}

// PlatformStats holds aggregate telemetry for the admin dashboard.
type PlatformStats struct {
	TotalUsers       int `json:"total_users"`
	TotalChallenges  int `json:"total_challenges"`
	TotalSubmissions int `json:"total_submissions"`
	TotalLessons     int `json:"total_lessons"`
	TotalSolves      int `json:"total_solves"`
	ActiveToday      int `json:"active_today"`
}

func (s *AdminService) GetPlatformStats(ctx context.Context) (*PlatformStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM challenges),
			(SELECT COUNT(*) FROM submissions),
			(SELECT COUNT(*) FROM lessons),
			(SELECT COUNT(*) FROM user_challenge_progress WHERE status = 'solved'),
			(SELECT COUNT(DISTINCT user_id) FROM submissions WHERE created_at >= CURRENT_DATE)`

	var stats PlatformStats
	err := s.pool.QueryRow(ctx, query).Scan(
		&stats.TotalUsers, &stats.TotalChallenges, &stats.TotalSubmissions,
		&stats.TotalLessons, &stats.TotalSolves, &stats.ActiveToday,
	)
	if err != nil {
		return nil, fmt.Errorf("getting platform stats: %w", err)
	}
	return &stats, nil
}

// CreateChallengeInput is the admin payload for creating a new challenge.
type CreateChallengeInput struct {
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Difficulty          int      `json:"difficulty"`
	LanguageSlug        string   `json:"language_slug"`
	VulnCategorySlug    string   `json:"vuln_category_slug"`
	VulnerableCode      string   `json:"vulnerable_code"`
	TargetVulnerability string   `json:"target_vulnerability"`
	ConceptualFix       string   `json:"conceptual_fix"`
	Hints               []string `json:"hints"`
	Points              int      `json:"points"`
	IsPublished         bool     `json:"is_published"`
}

func (s *AdminService) CreateChallenge(ctx context.Context, input CreateChallengeInput) (*model.Challenge, error) {
	lang, err := s.challengeRepo.GetLanguageBySlug(ctx, input.LanguageSlug)
	if err != nil {
		return nil, fmt.Errorf("invalid language: %w", err)
	}

	cat, err := s.challengeRepo.GetVulnCategoryBySlug(ctx, input.VulnCategorySlug)
	if err != nil {
		return nil, fmt.Errorf("invalid vulnerability category: %w", err)
	}

	if input.Points <= 0 {
		input.Points = 100
	}

	slug := generateSlug(input.Title)
	lineCount := strings.Count(input.VulnerableCode, "\n") + 1

	ch := &model.Challenge{
		ID:                  uuid.New(),
		Title:               input.Title,
		Slug:                slug,
		Description:         input.Description,
		Difficulty:          input.Difficulty,
		LanguageID:          lang.ID,
		VulnCategoryID:      cat.ID,
		VulnerableCode:      input.VulnerableCode,
		TargetVulnerability: input.TargetVulnerability,
		ConceptualFix:       input.ConceptualFix,
		Hints:               input.Hints,
		Points:              input.Points,
		LineCount:           lineCount,
		IsPublished:         input.IsPublished,
	}

	if err := s.challengeRepo.Insert(ctx, ch); err != nil {
		return nil, fmt.Errorf("creating challenge: %w", err)
	}

	ch.Language = lang
	ch.VulnCategory = cat

	return ch, nil
}

// UpdateChallengeInput is the admin payload for updating an existing challenge.
type UpdateChallengeInput struct {
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Difficulty          int      `json:"difficulty"`
	LanguageSlug        string   `json:"language_slug"`
	VulnCategorySlug    string   `json:"vuln_category_slug"`
	VulnerableCode      string   `json:"vulnerable_code"`
	TargetVulnerability string   `json:"target_vulnerability"`
	ConceptualFix       string   `json:"conceptual_fix"`
	Hints               []string `json:"hints"`
	Points              int      `json:"points"`
	IsPublished         bool     `json:"is_published"`
}

func (s *AdminService) UpdateChallenge(ctx context.Context, id uuid.UUID, input UpdateChallengeInput) (*model.Challenge, error) {
	lang, err := s.challengeRepo.GetLanguageBySlug(ctx, input.LanguageSlug)
	if err != nil {
		return nil, fmt.Errorf("invalid language: %w", err)
	}

	cat, err := s.challengeRepo.GetVulnCategoryBySlug(ctx, input.VulnCategorySlug)
	if err != nil {
		return nil, fmt.Errorf("invalid vulnerability category: %w", err)
	}

	if input.Points <= 0 {
		input.Points = 100
	}

	lineCount := strings.Count(input.VulnerableCode, "\n") + 1

	ch := &model.Challenge{
		ID:                  id,
		Title:               input.Title,
		Description:         input.Description,
		Difficulty:          input.Difficulty,
		LanguageID:          lang.ID,
		VulnCategoryID:      cat.ID,
		VulnerableCode:      input.VulnerableCode,
		TargetVulnerability: input.TargetVulnerability,
		ConceptualFix:       input.ConceptualFix,
		Hints:               input.Hints,
		Points:              input.Points,
		LineCount:           lineCount,
		IsPublished:         input.IsPublished,
	}

	if err := s.challengeRepo.Update(ctx, ch); err != nil {
		return nil, fmt.Errorf("updating challenge: %w", err)
	}

	ch.Language = lang
	ch.VulnCategory = cat

	return ch, nil
}

// CreateLessonInput is the admin payload for publishing a new lesson.
type CreateLessonInput struct {
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Difficulty  int      `json:"difficulty"`
	ReadTimeMin int      `json:"read_time_min"`
	Tags        []string `json:"tags"`
	IsPublished bool     `json:"is_published"`
}

func (s *AdminService) CreateLesson(ctx context.Context, input CreateLessonInput) (*model.Lesson, error) {
	if input.ReadTimeMin <= 0 {
		// Estimate reading time: ~200 words per minute
		wordCount := len(strings.Fields(input.Content))
		input.ReadTimeMin = wordCount/200 + 1
	}

	slug := generateSlug(input.Title)

	lesson := &model.Lesson{
		ID:          uuid.New(),
		Title:       input.Title,
		Slug:        slug,
		Category:    input.Category,
		Description: input.Description,
		Content:     input.Content,
		Difficulty:  input.Difficulty,
		ReadTimeMin: input.ReadTimeMin,
		Tags:        input.Tags,
		IsPublished: input.IsPublished,
	}

	if err := s.lessonRepo.Insert(ctx, lesson); err != nil {
		return nil, fmt.Errorf("creating lesson: %w", err)
	}

	return lesson, nil
}

// ListCommunityQueue returns community challenges filtered by status.
func (s *AdminService) ListCommunityQueue(ctx context.Context, status string, limit, offset int) ([]model.CommunityChallenge, int, error) {
	if status == "" {
		status = model.CommunityStatusPending
	}
	return s.communityRepo.ListByStatus(ctx, status, limit, offset)
}

// GetCommunityChallenge returns a single community challenge by ID.
func (s *AdminService) GetCommunityChallenge(ctx context.Context, id uuid.UUID) (*model.CommunityChallenge, error) {
	return s.communityRepo.GetByID(ctx, id)
}

// ReviewCommunityChallenge approves or rejects a community challenge.
func (s *AdminService) ReviewCommunityChallenge(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, action string, notes string) error {
	var status string
	switch action {
	case "approve":
		status = model.CommunityStatusApproved
	case "reject":
		status = model.CommunityStatusRejected
	default:
		return fmt.Errorf("invalid action: must be 'approve' or 'reject'")
	}
	return s.communityRepo.SetStatus(ctx, id, status, reviewerID, notes)
}

// PublishCommunityChallenge creates a real challenge from an approved community submission.
func (s *AdminService) PublishCommunityChallenge(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) (*model.Challenge, error) {
	cc, err := s.communityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting community challenge: %w", err)
	}
	if cc.Status != model.CommunityStatusApproved {
		return nil, fmt.Errorf("challenge must be approved before publishing")
	}

	// Create the real challenge using existing logic
	input := CreateChallengeInput{
		Title:               cc.Title,
		Description:         cc.Description,
		Difficulty:          cc.Difficulty,
		LanguageSlug:        cc.LanguageSlug,
		VulnCategorySlug:    cc.VulnCategorySlug,
		VulnerableCode:      cc.VulnerableCode,
		TargetVulnerability: cc.TargetVulnerability,
		ConceptualFix:       cc.ConceptualFix,
		Hints:               cc.Hints,
		Points:              cc.Points,
		IsPublished:         true,
	}

	challenge, err := s.CreateChallenge(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("publishing community challenge: %w", err)
	}

	// Link the community submission to the published challenge
	if err := s.communityRepo.SetChallengeID(ctx, id, challenge.ID); err != nil {
		return nil, fmt.Errorf("linking community challenge: %w", err)
	}

	return challenge, nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)
	// Collapse multiple dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}
