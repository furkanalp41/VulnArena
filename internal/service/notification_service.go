package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// WSBroadcaster is an interface for broadcasting WebSocket events.
// Avoids importing the websocket package directly (prevents circular deps).
type WSBroadcaster interface {
	BroadcastEvent(eventType string, payload map[string]string)
}

// NotificationService sends events to external channels (Discord webhooks, WebSockets).
// Safe to call even when disabled — methods are no-ops if webhookURL is empty.
type NotificationService struct {
	webhookURL string
	httpClient *http.Client
	logger     *slog.Logger
	wsHub      WSBroadcaster
}

func NewNotificationService(webhookURL string, wsHub WSBroadcaster, logger *slog.Logger) *NotificationService {
	return &NotificationService{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		wsHub:      wsHub,
	}
}

func (s *NotificationService) Enabled() bool {
	return s.webhookURL != ""
}

// SendFirstBlood fires a Discord embed announcing a First Blood event.
// Runs asynchronously — never blocks the caller.
func (s *NotificationService) SendFirstBlood(ctx context.Context, username, challengeTitle string, bonusXP int) {
	// WebSocket broadcast (always, regardless of Discord config)
	if s.wsHub != nil {
		s.wsHub.BroadcastEvent("FIRST_BLOOD", map[string]string{
			"user":      username,
			"challenge": challengeTitle,
			"bonus_xp":  fmt.Sprintf("%d", bonusXP),
		})
	}

	if !s.Enabled() {
		return
	}

	embed := discordEmbed{
		Title:       "\xF0\x9F\x9A\xA8 FIRST BLOOD!",
		Description: fmt.Sprintf("Hacker **%s** just pwned **%s** and secured the First Blood! (+%d bonus XP)", username, challengeTitle, bonusXP),
		Color:       0xFF4444,
		Footer:      &discordFooter{Text: "VulnArena // First Blood Tracker"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	go s.sendEmbed(embed)
}

// SendAchievementUnlocked fires a Discord embed announcing a badge unlock.
func (s *NotificationService) SendAchievementUnlocked(ctx context.Context, username, achievementName string) {
	// WebSocket broadcast (always)
	if s.wsHub != nil {
		s.wsHub.BroadcastEvent("ACHIEVEMENT", map[string]string{
			"user":        username,
			"achievement": achievementName,
		})
	}

	if !s.Enabled() {
		return
	}

	embed := discordEmbed{
		Title:       "\xF0\x9F\x8F\x85 ACHIEVEMENT UNLOCKED!",
		Description: fmt.Sprintf("Hacker **%s** earned the **%s** badge!", username, achievementName),
		Color:       0xFFD700,
		Footer:      &discordFooter{Text: "VulnArena // Achievement System"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	go s.sendEmbed(embed)
}

func (s *NotificationService) sendEmbed(embed discordEmbed) {
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal discord payload", slog.String("error", err.Error()))
		return
	}

	req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		s.logger.Error("failed to create discord request", slog.String("error", err.Error()))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("discord webhook failed", slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s.logger.Warn("discord webhook returned error", slog.Int("status", resp.StatusCode))
	}
}

// Discord webhook payload types
type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Footer      *discordFooter `json:"footer,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type discordFooter struct {
	Text string `json:"text"`
}
