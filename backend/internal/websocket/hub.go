package websocket

import (
	"encoding/json"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
	// Grace period before broadcasting participant_left to handle page reloads
	// Increased from 2s to 10s to handle high-latency networks (150ms+) and slow page loads
	disconnectGracePeriod = 10 * time.Second
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	UserID   uuid.UUID
	UserName string
	RoomID   string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
}

// PendingDisconnect tracks a user who disconnected but may reconnect (page reload)
type PendingDisconnect struct {
	UserID   uuid.UUID
	RoomID   string
	Timer    *time.Timer
	Canceled bool
}

// Hub manages WebSocket connections
type Hub struct {
	clients            map[*Client]bool
	rooms              map[string]map[*Client]bool
	register           chan *Client
	unregister         chan *Client
	broadcast          chan *RoomMessage
	mu                 sync.RWMutex
	pendingDisconnects map[string]*PendingDisconnect         // key: "roomID-userID"
	OnUserLeftRoom     func(roomID string, userID uuid.UUID) // Callback when user leaves room
}

// RoomMessage is a message to broadcast to a room
type RoomMessage struct {
	RoomID  string
	Message []byte
	Exclude *Client
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:            make(map[*Client]bool),
		rooms:              make(map[string]map[*Client]bool),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		broadcast:          make(chan *RoomMessage, 256),
		pendingDisconnects: make(map[string]*PendingDisconnect),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			slog.Debug("hub: registering client",
				"clientId", client.ID,
				"userId", client.UserID.String(),
				"userName", client.UserName,
				"roomId", client.RoomID,
			)
			h.mu.Lock()
			h.clients[client] = true
			if client.RoomID != "" {
				if h.rooms[client.RoomID] == nil {
					h.rooms[client.RoomID] = make(map[*Client]bool)
				}
				h.rooms[client.RoomID][client] = true

				// Cancel any pending disconnect for this user in this room (page reload case)
				pendingKey := client.RoomID + "-" + client.UserID.String()
				if pending, exists := h.pendingDisconnects[pendingKey]; exists {
					slog.Debug("hub: canceling pending disconnect (user reconnected)",
						"userId", client.UserID.String(),
						"roomId", client.RoomID,
					)
					pending.Canceled = true
					pending.Timer.Stop()
					delete(h.pendingDisconnects, pendingKey)
				}
			}
			slog.Debug("hub: client registered",
				"totalClients", len(h.clients),
				"roomClients", len(h.rooms[client.RoomID]),
			)
			h.mu.Unlock()

		case client := <-h.unregister:
			slog.Debug("hub: unregistering client",
				"clientId", client.ID,
				"userId", client.UserID.String(),
				"userName", client.UserName,
				"roomId", client.RoomID,
			)
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				roomID := client.RoomID
				userID := client.UserID
				delete(h.clients, client)
				slog.Debug("hub: client removed from clients map",
					"clientId", client.ID,
					"remainingClients", len(h.clients),
				)
				if roomID != "" {
					delete(h.rooms[roomID], client)
					slog.Debug("hub: client removed from room",
						"clientId", client.ID,
						"roomId", roomID,
						"remainingInRoom", len(h.rooms[roomID]),
					)
					// Check if user still has other connections in the room
					userStillInRoom := false
					for c := range h.rooms[roomID] {
						if c.UserID == userID {
							userStillInRoom = true
							break
						}
					}
					slog.Debug("hub: checking if user still in room",
						"userId", userID.String(),
						"roomId", roomID,
						"userStillInRoom", userStillInRoom,
					)
					if len(h.rooms[roomID]) == 0 {
						delete(h.rooms, roomID)
					}
					// Schedule participant_left broadcast with grace period (for page reload handling)
					if !userStillInRoom && roomID != "" {
						pendingKey := roomID + "-" + userID.String()
						// Only schedule if not already pending
						if _, exists := h.pendingDisconnects[pendingKey]; !exists {
							slog.Debug("hub: scheduling participant_left with grace period",
								"userId", userID.String(),
								"roomId", roomID,
								"gracePeriod", disconnectGracePeriod,
							)
							pending := &PendingDisconnect{
								UserID:   userID,
								RoomID:   roomID,
								Canceled: false,
							}
							h.pendingDisconnects[pendingKey] = pending

							// Start timer for delayed broadcast
							pending.Timer = time.AfterFunc(disconnectGracePeriod, func() {
								h.mu.Lock()
								// Check if still pending (not canceled by reconnection)
								if p, exists := h.pendingDisconnects[pendingKey]; exists && !p.Canceled {
									delete(h.pendingDisconnects, pendingKey)
									// Double-check user hasn't reconnected
									stillInRoom := false
									if room, roomExists := h.rooms[roomID]; roomExists {
										for c := range room {
											if c.UserID == userID {
												stillInRoom = true
												break
											}
										}
									}
									h.mu.Unlock()

									if !stillInRoom {
										slog.Debug("hub: grace period expired, broadcasting participant_left",
											"userId", userID.String(),
											"roomId", roomID,
										)
										h.BroadcastToRoom(roomID, Message{
											Type: "participant_left",
											Payload: map[string]interface{}{
												"userId": userID,
											},
										})
										// Call callback if set (for team_members_updated broadcast)
										if h.OnUserLeftRoom != nil {
											slog.Debug("hub: calling OnUserLeftRoom callback",
												"roomId", roomID,
												"userId", userID.String(),
											)
											h.OnUserLeftRoom(roomID, userID)
										}
									} else {
										slog.Debug("hub: user reconnected during grace period, skipping broadcast",
											"userId", userID.String(),
											"roomId", roomID,
										)
									}
								} else {
									h.mu.Unlock()
									slog.Debug("hub: pending disconnect was canceled",
										"userId", userID.String(),
										"roomId", roomID,
									)
								}
							})
						}
					}
				} else {
					slog.Debug("hub: roomID is empty, skipping broadcast",
						"clientId", client.ID,
						"userId", userID.String(),
					)
				}
				close(client.Send)
			} else {
				slog.Warn("hub: unregister called but client not in clients map",
					"clientId", client.ID,
					"userId", client.UserID.String(),
					"roomId", client.RoomID,
				)
				// Still try to remove from room in case it was added via JoinRoom
				if client.RoomID != "" {
					if _, exists := h.rooms[client.RoomID]; exists {
						delete(h.rooms[client.RoomID], client)
						slog.Debug("hub: orphan client removed from room",
							"clientId", client.ID,
							"roomId", client.RoomID,
						)
					}
				}
			}
			h.mu.Unlock()

		case roomMsg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.rooms[roomMsg.RoomID]; ok {
				clientCount := len(clients)
				slog.Debug("hub: broadcasting to room",
					"roomID", roomMsg.RoomID,
					"clientCount", clientCount,
				)
				for client := range clients {
					if roomMsg.Exclude != nil && client == roomMsg.Exclude {
						continue
					}
					select {
					case client.Send <- roomMsg.Message:
						slog.Debug("hub: message sent to client",
							"roomID", roomMsg.RoomID,
							"clientID", client.ID,
						)
					default:
						slog.Warn("hub: client send channel full, removing client",
							"roomID", roomMsg.RoomID,
							"clientID", client.ID,
						)
						h.mu.RUnlock()
						h.mu.Lock()
						delete(h.clients, client)
						delete(h.rooms[roomMsg.RoomID], client)
						close(client.Send)
						h.mu.Unlock()
						h.mu.RLock()
					}
				}
			} else {
				slog.Warn("hub: room not found for broadcast",
					"roomID", roomMsg.RoomID,
				)
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToRoom broadcasts a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	h.broadcast <- &RoomMessage{RoomID: roomID, Message: data}
}

// BroadcastToRoomExcept broadcasts a message to all clients in a room except one
func (h *Hub) BroadcastToRoomExcept(roomID string, msg Message, exclude *Client) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	h.broadcast <- &RoomMessage{RoomID: roomID, Message: data, Exclude: exclude}
}

// JoinRoom moves a client to a room
func (h *Hub) JoinRoom(client *Client, roomID string) {
	slog.Debug("hub: client joining room",
		"clientId", client.ID,
		"userId", client.UserID.String(),
		"userName", client.UserName,
		"fromRoom", client.RoomID,
		"toRoom", roomID,
	)
	h.mu.Lock()
	defer h.mu.Unlock()

	// Leave current room
	if client.RoomID != "" {
		delete(h.rooms[client.RoomID], client)
		if len(h.rooms[client.RoomID]) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}

	// Join new room
	client.RoomID = roomID
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	slog.Debug("hub: client joined room",
		"roomId", roomID,
		"roomClientCount", len(h.rooms[roomID]),
	)
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.RoomID != "" {
		delete(h.rooms[client.RoomID], client)
		if len(h.rooms[client.RoomID]) == 0 {
			delete(h.rooms, client.RoomID)
		}
		client.RoomID = ""
	}
}

// GetRoomClients returns unique users in a room (deduplicated by UserID)
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	seen := make(map[uuid.UUID]bool)
	clients := make([]*Client, 0)
	if room, ok := h.rooms[roomID]; ok {
		for client := range room {
			if !seen[client.UserID] {
				seen[client.UserID] = true
				clients = append(clients, client)
			}
		}
	}
	return clients
}

// IsUserInRoom checks if a user is already in a room
func (h *Hub) IsUserInRoom(roomID string, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.rooms[roomID]; ok {
		for client := range room {
			if client.UserID == userID {
				return true
			}
		}
	}
	return false
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(client *Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	select {
	case client.Send <- data:
	default:
		log.Printf("Client send buffer full, dropping message")
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump(handler func(*Client, []byte)) {
	defer func() {
		c.Hub.Unregister(c)
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		handler(c, message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
