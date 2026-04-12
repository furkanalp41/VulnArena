package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 2048
)

func newUpgrader(allowedOrigins map[string]bool) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return allowedOrigins[origin]
		},
	}
}

// ----- Inbound message types (client → server) -----

const (
	MsgJoinRoom   = "JOIN_ROOM"
	MsgLeaveRoom  = "LEAVE_ROOM"
	MsgCursorMove = "CURSOR_MOVE"
	MsgLineSelect = "LINE_SELECT"
)

// ----- Outbound event types (server → client) -----

const (
	EventRoomUserJoined   = "ROOM_USER_JOINED"
	EventRoomUserLeft     = "ROOM_USER_LEFT"
	EventRemoteCursor     = "REMOTE_CURSOR"
	EventRemoteLineSelect = "REMOTE_LINE_SELECT"
	EventRoomState        = "ROOM_STATE"
)

// InboundMessage is the envelope for all client-to-server WebSocket messages.
type InboundMessage struct {
	Type        string `json:"type"`
	ChallengeID string `json:"challenge_id,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Selected    *bool  `json:"selected,omitempty"`
}

// RoomMessage carries a message scoped to a single room.
type RoomMessage struct {
	RoomKey    string
	Data       []byte
	SenderOnly *Client // nil = broadcast to all; non-nil = skip this client
}

// RoomJoin is the internal signal to move a client into a room.
type RoomJoin struct {
	Client  *Client
	RoomKey string
}

// Hub maintains the set of active WebSocket clients and broadcasts messages.
type Hub struct {
	clients        map[*Client]bool
	rooms          map[string]map[*Client]bool
	broadcast      chan []byte
	roomMsg        chan RoomMessage
	register       chan *Client
	unregister     chan *Client
	joinRoom       chan RoomJoin
	leaveRoom      chan *Client
	mu             sync.RWMutex
	logger         *slog.Logger
	allowedOrigins map[string]bool
}

// Client represents a single WebSocket connection.
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	UserID      string
	Username    string
	DisplayName string
	RoomKey     string // "" = global-only, "challengeID:teamID" = in a room
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger, allowedOrigins []string) *Hub {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = true
	}
	return &Hub{
		clients:        make(map[*Client]bool),
		rooms:          make(map[string]map[*Client]bool),
		broadcast:      make(chan []byte, 256),
		roomMsg:        make(chan RoomMessage, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		joinRoom:       make(chan RoomJoin),
		leaveRoom:      make(chan *Client),
		logger:         logger,
		allowedOrigins: origins,
	}
}

// Run starts the hub's main event loop. Should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("ws client connected", slog.Int("total", len(h.clients)))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				// Remove from room if in one
				if client.RoomKey != "" {
					if room, exists := h.rooms[client.RoomKey]; exists {
						delete(room, client)
						if len(room) == 0 {
							delete(h.rooms, client.RoomKey)
						} else {
							// Notify remaining room members
							h.broadcastToRoomLocked(client.RoomKey, client, buildRoomEvent(EventRoomUserLeft, client))
						}
					}
				}
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("ws client disconnected", slog.Int("total", len(h.clients)))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case join := <-h.joinRoom:
			h.mu.Lock()
			client := join.Client

			// Leave existing room first
			if client.RoomKey != "" && client.RoomKey != join.RoomKey {
				if room, exists := h.rooms[client.RoomKey]; exists {
					delete(room, client)
					if len(room) == 0 {
						delete(h.rooms, client.RoomKey)
					} else {
						h.broadcastToRoomLocked(client.RoomKey, client, buildRoomEvent(EventRoomUserLeft, client))
					}
				}
			}

			// Join new room
			client.RoomKey = join.RoomKey
			if h.rooms[join.RoomKey] == nil {
				h.rooms[join.RoomKey] = make(map[*Client]bool)
			}
			h.rooms[join.RoomKey][client] = true

			// Send current room state to the joining client
			h.sendRoomStateLocked(client, join.RoomKey)

			// Notify existing room members
			h.broadcastToRoomLocked(join.RoomKey, client, buildRoomEvent(EventRoomUserJoined, client))
			h.mu.Unlock()

			h.logger.Debug("client joined room",
				slog.String("room", join.RoomKey),
				slog.String("user", client.Username),
				slog.Int("room_size", len(h.rooms[join.RoomKey])))

		case client := <-h.leaveRoom:
			h.mu.Lock()
			if client.RoomKey != "" {
				if room, exists := h.rooms[client.RoomKey]; exists {
					delete(room, client)
					if len(room) == 0 {
						delete(h.rooms, client.RoomKey)
					} else {
						h.broadcastToRoomLocked(client.RoomKey, client, buildRoomEvent(EventRoomUserLeft, client))
					}
				}
				client.RoomKey = ""
			}
			h.mu.Unlock()

		case msg := <-h.roomMsg:
			h.mu.RLock()
			if room, exists := h.rooms[msg.RoomKey]; exists {
				for client := range room {
					if msg.SenderOnly != nil && client == msg.SenderOnly {
						continue
					}
					select {
					case client.send <- msg.Data:
					default:
						close(client.send)
						delete(h.clients, client)
						delete(room, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// broadcastToRoomLocked sends a message to all clients in a room except the sender.
// Must be called while holding h.mu.
func (h *Hub) broadcastToRoomLocked(roomKey string, sender *Client, data []byte) {
	room, exists := h.rooms[roomKey]
	if !exists {
		return
	}
	for client := range room {
		if client == sender {
			continue
		}
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
			delete(room, client)
		}
	}
}

// sendRoomStateLocked sends the current list of room members to a client.
// Must be called while holding h.mu.
func (h *Hub) sendRoomStateLocked(client *Client, roomKey string) {
	room, exists := h.rooms[roomKey]
	if !exists {
		return
	}

	members := make([]map[string]string, 0, len(room))
	for c := range room {
		if c == client {
			continue
		}
		members = append(members, map[string]string{
			"user_id":      c.UserID,
			"username":     c.Username,
			"display_name": c.DisplayName,
		})
	}

	payload := map[string]any{
		"type":    EventRoomState,
		"room":    roomKey,
		"members": members,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

// Broadcast sends a raw message to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// BroadcastEvent marshals a typed event and broadcasts it.
func (h *Hub) BroadcastEvent(eventType string, payload map[string]string) {
	data := make(map[string]string, len(payload)+1)
	for k, v := range payload {
		data[k] = v
	}
	data["type"] = eventType

	msg, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("failed to marshal ws event", slog.String("error", err.Error()))
		return
	}
	h.broadcast <- msg
}

// BroadcastToRoom sends a message to all clients in a specific room, optionally skipping the sender.
func (h *Hub) BroadcastToRoom(roomKey string, sender *Client, msg []byte) {
	h.roomMsg <- RoomMessage{
		RoomKey:    roomKey,
		Data:       msg,
		SenderOnly: sender,
	}
}

// GetRoomClients returns a snapshot of clients in a room.
func (h *Hub) GetRoomClients(roomKey string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomKey]
	if !exists {
		return nil
	}

	clients := make([]*Client, 0, len(room))
	for c := range room {
		clients = append(clients, c)
	}
	return clients
}

// buildRoomEvent creates a JSON payload for room membership events.
func buildRoomEvent(eventType string, client *Client) []byte {
	payload := map[string]string{
		"type":         eventType,
		"user_id":      client.UserID,
		"username":     client.Username,
		"display_name": client.DisplayName,
	}
	data, _ := json.Marshal(payload)
	return data
}

// ServeWS handles WebSocket upgrade requests (unauthenticated, for global notifications).
func ServeWS(hub *Hub) http.HandlerFunc {
	up := newUpgrader(hub.allowedOrigins)
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			hub.logger.Error("ws upgrade failed", slog.String("error", err.Error()))
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

// readPump reads messages from the WebSocket connection.
// For authenticated collab clients, it dispatches inbound messages.
// For unauthenticated clients, it only handles keepalive.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Unauthenticated clients have no UserID — ignore their messages
		if c.UserID == "" {
			continue
		}

		var msg InboundMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleInbound(msg)
	}
}

// handleInbound dispatches parsed inbound messages.
func (c *Client) handleInbound(msg InboundMessage) {
	switch msg.Type {
	case MsgJoinRoom:
		if msg.ChallengeID == "" || msg.TeamID == "" {
			return
		}
		roomKey := msg.ChallengeID + ":" + msg.TeamID
		c.hub.joinRoom <- RoomJoin{Client: c, RoomKey: roomKey}

	case MsgLeaveRoom:
		c.hub.leaveRoom <- c

	case MsgCursorMove:
		if c.RoomKey == "" {
			return
		}
		payload := map[string]any{
			"type":         EventRemoteCursor,
			"user_id":      c.UserID,
			"username":     c.Username,
			"display_name": c.DisplayName,
			"line":         msg.Line,
			"column":       msg.Column,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		c.hub.BroadcastToRoom(c.RoomKey, c, data)

	case MsgLineSelect:
		if c.RoomKey == "" || msg.Selected == nil {
			return
		}
		payload := map[string]any{
			"type":         EventRemoteLineSelect,
			"user_id":      c.UserID,
			"username":     c.Username,
			"display_name": c.DisplayName,
			"line":         msg.Line,
			"selected":     *msg.Selected,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		c.hub.BroadcastToRoom(c.RoomKey, c, data)
	}
}

// writePump writes messages from the send channel to the WebSocket.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages into the same write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
