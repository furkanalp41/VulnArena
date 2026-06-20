package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	Env               string
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	RedisURL          string
	JWT               JWTConfig
	Anthropic         AnthropicConfig
	DiscordWebhookURL string
	AllowedOrigins    []string
}

type AnthropicConfig struct {
	APIKey string
	Model  string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("PORT", "8080"),
		Env:      getEnv("ENV", "development"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}

	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	cfg.DatabaseURL = dbURL

	maxConns, err := parseInt32Env("DB_MAX_CONNS", 25)
	if err != nil {
		return nil, err
	}
	cfg.DBMaxConns = maxConns

	minConns, err := parseInt32Env("DB_MIN_CONNS", 5)
	if err != nil {
		return nil, err
	}
	cfg.DBMinConns = minConns
	if cfg.DBMinConns > cfg.DBMaxConns {
		return nil, fmt.Errorf("DB_MIN_CONNS (%d) cannot exceed DB_MAX_CONNS (%d)",
			cfg.DBMinConns, cfg.DBMaxConns)
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	weakSecrets := map[string]bool{
		"change-me":        true,
		"changeme":         true,
		"CHANGE_ME":        true,
		"your-secret-here": true,
		"secret":           true,
	}
	if weakSecrets[jwtSecret] {
		return nil, fmt.Errorf("JWT_SECRET must not be a placeholder value")
	}
	cfg.JWT.Secret = jwtSecret

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	cfg.JWT.AccessTTL = accessTTL

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	cfg.JWT.RefreshTTL = refreshTTL

	// Anthropic (optional — falls back to keyword evaluator if absent)
	cfg.Anthropic = AnthropicConfig{
		APIKey: getEnv("ANTHROPIC_API_KEY", ""),
		Model:  getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
	}

	// Discord webhook (optional — disabled if empty)
	cfg.DiscordWebhookURL = getEnv("DISCORD_WEBHOOK_URL", "")

	// CORS allowed origins — dev defaults are Vite/SvelteKit local ports.
	defaultOrigins := "http://localhost:5173,http://localhost:4173"
	originsRaw := getEnv("ALLOWED_ORIGINS", defaultOrigins)
	cfg.AllowedOrigins = splitAndTrim(originsRaw)

	return cfg, nil
}

func parseInt32Env(key string, fallback int32) (int32, error) {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be > 0, got %d", key, n)
	}
	return int32(n), nil
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
