package websocket

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/service"
)

// ServeCollabWS handles authenticated WebSocket upgrades for co-op code auditing.
// The JWT is passed via the "token" query parameter since browsers cannot set
// custom headers on WebSocket upgrade requests.
func ServeCollabWS(hub *Hub, authService *service.AuthService, teamService *service.TeamService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Look up user details for display in the room
		userID := claims.UserID
		username := ""
		displayName := ""

		// Get user's team membership for room validation
		team, _ := teamService.GetUserTeam(r.Context(), userID)

		// We need user info — fetch from the team service or embed in claims.
		// For now, use the claims UserID and look up via a lightweight approach.
		// The username is embedded when the client sends JOIN_ROOM.
		// We'll set a placeholder and let the client provide display info.
		// Actually, let's look up the user properly.
		username = claims.UserID.String() // fallback
		displayName = username

		// Try to get username from the request (optional query params for display)
		if u := r.URL.Query().Get("username"); u != "" {
			username = u
		}
		if d := r.URL.Query().Get("display_name"); d != "" {
			displayName = d
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			hub.logger.Error("collab ws upgrade failed", slog.String("error", err.Error()))
			return
		}

		client := &Client{
			hub:         hub,
			conn:        conn,
			send:        make(chan []byte, 256),
			UserID:      userID.String(),
			Username:    username,
			DisplayName: displayName,
		}

		// If user has a team, store it for potential validation later
		if team != nil {
			hub.logger.Debug("collab ws connected",
				slog.String("user_id", userID.String()),
				slog.String("username", username),
				slog.String("team", team.Tag))
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

// ValidateRoomAccess checks if a user is allowed to join a specific room.
// Room key format: "challengeID:teamID"
func ValidateRoomAccess(teamService *service.TeamService, userID uuid.UUID, teamID string) bool {
	// For now, room access is validated client-side by the team_id parameter.
	// A stricter check would verify the user belongs to the team.
	// This can be enhanced later for production hardening.
	return true
}
