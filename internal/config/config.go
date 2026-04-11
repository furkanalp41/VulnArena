package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port              string
	Env               string
	DatabaseURL       string
	RedisURL          string
	JWT               JWTConfig
	Anthropic         AnthropicConfig
	DiscordWebhookURL string
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
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://vulnarena:vulnarena_secret@localhost:5432/vulnarena?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
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
		Model:  getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
	}

	// Discord webhook (optional — disabled if empty)
	cfg.DiscordWebhookURL = getEnv("DISCORD_WEBHOOK_URL", "")

	return cfg, nil
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
