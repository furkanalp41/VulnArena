package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/vulnarena/vulnarena/internal/handler"
	"github.com/vulnarena/vulnarena/internal/server/middleware"
	ws "github.com/vulnarena/vulnarena/internal/server/websocket"
	"github.com/vulnarena/vulnarena/internal/service"
)

type RouterDeps struct {
	Logger             *slog.Logger
	RedisClient        *redis.Client
	AuthService        *service.AuthService
	APIKeyService      *service.APIKeyService
	TeamService        *service.TeamService
	AuthHandler        *handler.AuthHandler
	UserHandler        *handler.UserHandler
	ArenaHandler       *handler.ArenaHandler
	AcademyHandler     *handler.AcademyHandler
	TelemetryHandler   *handler.TelemetryHandler
	AdminHandler       *handler.AdminHandler
	AchievementHandler *handler.AchievementHandler
	TeamHandler        *handler.TeamHandler
	APIKeyHandler      *handler.APIKeyHandler
	CommunityHandler   *handler.CommunityHandler
	WSHub              *ws.Hub
	AllowedOrigins     []string
}

func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Security)
	r.Use(middleware.Logging(deps.Logger))
	r.Use(middleware.CORS(deps.AllowedOrigins...))
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RateLimit(deps.RedisClient, 100, 1*time.Minute))

	// Health check
	r.Get("/health", handler.HealthCheck)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (strict rate limit: 10 req/min per IP, audited)
		r.Route("/auth", func(r chi.Router) {
			r.Use(middleware.StrictRateLimit(deps.RedisClient, 10, 1*time.Minute))
			r.Use(middleware.AuditLog(deps.Logger, "auth"))
			r.Post("/register", deps.AuthHandler.Register)
			r.Post("/login", deps.AuthHandler.Login)
			r.Post("/refresh", deps.AuthHandler.Refresh)
			r.Post("/logout", deps.AuthHandler.Logout)
		})

		// Public arena routes (browsing challenges)
		r.Route("/arena/challenges", func(r chi.Router) {
			r.Get("/", deps.ArenaHandler.ListChallenges)
			r.Get("/{id}", deps.ArenaHandler.GetChallenge)
		})

		// Public academy routes
		r.Route("/academy/lessons", func(r chi.Router) {
			r.Get("/", deps.AcademyHandler.ListLessons)
			r.Get("/{id}", deps.AcademyHandler.GetLesson)
		})

		// Public achievements catalog
		r.Get("/achievements", deps.AchievementHandler.ListAchievements)

		// Public leaderboard
		r.Get("/leaderboard", deps.TelemetryHandler.GetLeaderboard)

		// Public team routes (leaderboard BEFORE {tag} to avoid param capture)
		r.Route("/teams", func(r chi.Router) {
			r.Get("/", deps.TeamHandler.ListTeams)
			r.Get("/leaderboard", deps.TeamHandler.GetTeamLeaderboard)
			r.Get("/{tag}", deps.TeamHandler.GetTeam)
		})

		// WebSocket endpoints
		r.Get("/ws", ws.ServeWS(deps.WSHub))
		r.Get("/ws/collab", ws.ServeCollabWS(deps.WSHub, deps.AuthService, deps.TeamService))

		// Public user profiles
		r.Get("/users/{username}/profile", deps.TelemetryHandler.GetPublicProfile)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(deps.AuthService, deps.APIKeyService))

			r.Route("/users", func(r chi.Router) {
				r.Get("/me", deps.UserHandler.GetMe)
				r.Patch("/me", deps.UserHandler.UpdateMe)
			})

			// API key management (strict rate limit: 5 req/min, audited)
			r.Route("/user/api-key", func(r chi.Router) {
				r.Use(middleware.StrictRateLimit(deps.RedisClient, 5, 1*time.Minute))
				r.Use(middleware.AuditLog(deps.Logger, "api_key"))
				r.Post("/", deps.APIKeyHandler.Generate)
				r.Get("/", deps.APIKeyHandler.GetInfo)
				r.Delete("/", deps.APIKeyHandler.Revoke)
			})

			// Arena submissions (authenticated, audited)
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuditLog(deps.Logger, "arena"))
				r.Post("/arena/challenges/{id}/submit", deps.ArenaHandler.SubmitAnswer)
			})
			r.Get("/arena/challenges/{id}/submissions", deps.ArenaHandler.GetSubmissionHistory)

			// Community Forge (authenticated)
			r.Route("/community/challenges", func(r chi.Router) {
				r.Get("/", deps.CommunityHandler.ListMyChallenges)
				r.Post("/", deps.CommunityHandler.SubmitChallenge)
				r.Get("/{id}", deps.CommunityHandler.GetChallenge)
				r.Put("/{id}", deps.CommunityHandler.UpdateChallenge)
				r.Delete("/{id}", deps.CommunityHandler.DeleteChallenge)
			})

			// Dashboard telemetry
			r.Get("/dashboard/profile", deps.TelemetryHandler.GetDashboardProfile)

			// Team management (authenticated)
			r.Get("/teams/me", deps.TeamHandler.GetMyTeam)
			r.Post("/teams", deps.TeamHandler.CreateTeam)
			r.Post("/teams/leave", deps.TeamHandler.LeaveTeam)
			r.Post("/teams/{tag}/join", deps.TeamHandler.JoinTeam)

			// Admin routes (requires admin role, audited)
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.AdminOnly)
				r.Use(middleware.AuditLog(deps.Logger, "admin"))
				r.Get("/stats", deps.AdminHandler.GetPlatformStats)
				r.Post("/challenges", deps.AdminHandler.CreateChallenge)
				r.Put("/challenges/{id}", deps.AdminHandler.UpdateChallenge)
				r.Post("/lessons", deps.AdminHandler.CreateLesson)

				// Community Forge review
				r.Get("/community/queue", deps.AdminHandler.ListCommunityQueue)
				r.Get("/community/challenges/{id}", deps.AdminHandler.GetCommunityChallenge)
				r.Post("/community/challenges/{id}/review", deps.AdminHandler.ReviewCommunityChallenge)
				r.Post("/community/challenges/{id}/publish", deps.AdminHandler.PublishCommunityChallenge)
			})
		})
	})

	return r
}
