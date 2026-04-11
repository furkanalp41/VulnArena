package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/vulnarena/vulnarena/internal/config"
	"github.com/vulnarena/vulnarena/internal/database"
	"github.com/vulnarena/vulnarena/internal/handler"
	"github.com/vulnarena/vulnarena/internal/nlp"
	"github.com/vulnarena/vulnarena/internal/repository"
	"github.com/vulnarena/vulnarena/internal/server"
	ws "github.com/vulnarena/vulnarena/internal/server/websocket"
	"github.com/vulnarena/vulnarena/internal/service"
)

func main() {
	// Load .env file in development
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	ctx := context.Background()

	// Database connections
	pool, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to PostgreSQL")

	redisClient, err := database.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("connected to Redis")

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	challengeRepo := repository.NewChallengeRepository(pool)
	submissionRepo := repository.NewSubmissionRepository(pool)
	lessonRepo := repository.NewLessonRepository(pool)
	telemetryRepo := repository.NewTelemetryRepository(pool)
	achievementRepo := repository.NewAchievementRepository(pool)
	teamRepo := repository.NewTeamRepository(pool)

	// NLP evaluator: use Anthropic if API key is set, otherwise keyword-based fallback
	var evaluator nlp.SemanticMatcher
	if cfg.Anthropic.APIKey != "" {
		evaluator = nlp.NewAnthropicEvaluator(cfg.Anthropic.APIKey, cfg.Anthropic.Model)
		logger.Info("semantic evaluator: Anthropic API", slog.String("model", cfg.Anthropic.Model))
	} else {
		evaluator = nlp.NewLLMEvaluator(800 * time.Millisecond)
		logger.Info("semantic evaluator: keyword-based (set ANTHROPIC_API_KEY for AI evaluation)")
	}

	// WebSocket hub
	wsHub := ws.NewHub(logger)
	go wsHub.Run()

	// Services
	authService := service.NewAuthService(userRepo, redisClient, cfg.JWT)
	userService := service.NewUserService(userRepo)
	apiKeyService := service.NewAPIKeyService(userRepo)
	notificationService := service.NewNotificationService(cfg.DiscordWebhookURL, wsHub, logger)
	achievementService := service.NewAchievementService(achievementRepo, notificationService, logger)
	arenaService := service.NewArenaService(challengeRepo, submissionRepo, userRepo, evaluator, achievementService, notificationService, logger)
	academyService := service.NewAcademyService(lessonRepo)
	telemetryService := service.NewTelemetryService(telemetryRepo, userRepo, achievementRepo, redisClient)
	teamService := service.NewTeamService(teamRepo, logger)
	communityRepo := repository.NewCommunityRepository(pool)
	adminService := service.NewAdminService(pool, challengeRepo, lessonRepo, communityRepo)

	if notificationService.Enabled() {
		logger.Info("discord webhook: enabled")
	} else {
		logger.Info("discord webhook: disabled (set DISCORD_WEBHOOK_URL to enable)")
	}

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	arenaHandler := handler.NewArenaHandler(arenaService)
	academyHandler := handler.NewAcademyHandler(academyService)
	telemetryHandler := handler.NewTelemetryHandler(telemetryService)
	adminHandler := handler.NewAdminHandler(adminService)
	achievementHandler := handler.NewAchievementHandler(achievementService)
	teamHandler := handler.NewTeamHandler(teamService)

	// Community Forge
	communityService := service.NewCommunityService(communityRepo, telemetryRepo, logger)
	communityHandler := handler.NewCommunityHandler(communityService)

	// Router
	router := server.NewRouter(server.RouterDeps{
		Logger:             logger,
		RedisClient:        redisClient,
		AuthService:        authService,
		APIKeyService:      apiKeyService,
		TeamService:        teamService,
		AuthHandler:        authHandler,
		UserHandler:        userHandler,
		ArenaHandler:       arenaHandler,
		AcademyHandler:     academyHandler,
		TelemetryHandler:   telemetryHandler,
		AdminHandler:       adminHandler,
		AchievementHandler: achievementHandler,
		TeamHandler:        teamHandler,
		APIKeyHandler:      apiKeyHandler,
		CommunityHandler:   communityHandler,
		WSHub:              wsHub,
	})

	// Start server
	srv := server.New(router, cfg.Port, logger)
	if err := srv.Start(); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
