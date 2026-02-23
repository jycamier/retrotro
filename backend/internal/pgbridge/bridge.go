package pgbridge

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

const (
	channelRoom     = "retrotro_room"
	channelPresence = "retrotro_presence"
)

// RemoteUser represents a user connected on another pod
type RemoteUser struct {
	UserID   uuid.UUID
	UserName string
	PodID    string
}

// roomMessage is the envelope published via NOTIFY for room broadcasts
type roomMessage struct {
	PodID   string          `json:"podId"`
	RoomID  string          `json:"roomId"`
	Message json.RawMessage `json:"message"`
}

// presenceMessage is the envelope published via NOTIFY for presence events
type presenceMessage struct {
	PodID    string    `json:"podId"`
	RoomID   string    `json:"roomId"`
	UserID   uuid.UUID `json:"userId"`
	UserName string    `json:"userName,omitempty"`
	Action   string    `json:"action"` // "join" or "leave"
}

// PGBridge wraps the Hub and relays broadcasts between pods via PostgreSQL LISTEN/NOTIFY
type PGBridge struct {
	hub         *websocket.Hub
	pool        *pgxpool.Pool
	podID       string
	mu          sync.RWMutex
	remoteUsers map[string]map[string]RemoteUser // roomID -> userID -> info
	cancel      context.CancelFunc
}

// NewPGBridge creates a new PGBridge
func NewPGBridge(hub *websocket.Hub, pool *pgxpool.Pool) *PGBridge {
	return &PGBridge{
		hub:         hub,
		pool:        pool,
		podID:       uuid.New().String(),
		remoteUsers: make(map[string]map[string]RemoteUser),
	}
}

// Start begins listening on PostgreSQL channels
func (b *PGBridge) Start(_ context.Context) error {
	listenCtx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	conn, err := b.pool.Acquire(listenCtx)
	if err != nil {
		return err
	}

	// LISTEN on both channels
	_, err = conn.Exec(listenCtx, "LISTEN "+channelRoom)
	if err != nil {
		conn.Release()
		return err
	}
	_, err = conn.Exec(listenCtx, "LISTEN "+channelPresence)
	if err != nil {
		conn.Release()
		return err
	}

	slog.Info("pgbridge: listening on PostgreSQL channels",
		"podId", b.podID,
		"channels", []string{channelRoom, channelPresence},
	)

	// Listen loop in goroutine
	go func() {
		defer conn.Release()
		for {
			notification, err := conn.Conn().WaitForNotification(listenCtx)
			if err != nil {
				if listenCtx.Err() != nil {
					slog.Info("pgbridge: listener stopped (context canceled)")
					return
				}
				slog.Error("pgbridge: error waiting for notification", "error", err)
				return
			}

			switch notification.Channel {
			case channelRoom:
				b.handleRoomNotification(notification.Payload)
			case channelPresence:
				b.handlePresenceNotification(notification.Payload)
			}
		}
	}()

	return nil
}

// Stop cancels the listener
func (b *PGBridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}

// Hub returns the underlying Hub for local-only operations
func (b *PGBridge) Hub() *websocket.Hub {
	return b.hub
}

// BroadcastToRoom broadcasts locally and publishes to PostgreSQL
func (b *PGBridge) BroadcastToRoom(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("pgbridge: failed to marshal message", "error", err)
		return
	}

	// Broadcast locally
	b.hub.BroadcastRaw(roomID, data)

	// Publish to PostgreSQL
	b.publishRoom(roomID, data)
}

// BroadcastToRoomExcept broadcasts locally with exclusion and publishes to PostgreSQL
func (b *PGBridge) BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client) {
	// Local broadcast with exclusion
	b.hub.BroadcastToRoomExcept(roomID, msg, exclude)

	// Publish to PostgreSQL (no exclusion for remote pods, they don't have this client)
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("pgbridge: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

// GetRoomClients returns local + remote users in a room
func (b *PGBridge) GetRoomClients(roomID string) []*websocket.Client {
	localClients := b.hub.GetRoomClients(roomID)

	b.mu.RLock()
	remoteRoom, exists := b.remoteUsers[roomID]
	b.mu.RUnlock()

	if !exists || len(remoteRoom) == 0 {
		return localClients
	}

	// Collect local user IDs
	localUserIDs := make(map[uuid.UUID]bool, len(localClients))
	for _, c := range localClients {
		localUserIDs[c.UserID] = true
	}

	// Add remote users not already local
	for _, ru := range remoteRoom {
		if !localUserIDs[ru.UserID] {
			localClients = append(localClients, &websocket.Client{
				UserID:   ru.UserID,
				UserName: ru.UserName,
			})
		}
	}

	return localClients
}

// IsUserInRoom checks local + remote presence
func (b *PGBridge) IsUserInRoom(roomID string, userID uuid.UUID) bool {
	if b.hub.IsUserInRoom(roomID, userID) {
		return true
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if room, exists := b.remoteUsers[roomID]; exists {
		if _, ok := room[userID.String()]; ok {
			return true
		}
	}
	return false
}

// PublishPresenceJoin publishes a join event to other pods
func (b *PGBridge) PublishPresenceJoin(roomID string, userID uuid.UUID, userName string) {
	// Clean up remote users entry if user is now local
	b.mu.Lock()
	if room, exists := b.remoteUsers[roomID]; exists {
		delete(room, userID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, roomID)
		}
	}
	b.mu.Unlock()

	b.publishPresence(roomID, userID, userName, "join")
}

// PublishPresenceLeave publishes a leave event to other pods
func (b *PGBridge) PublishPresenceLeave(roomID string, userID uuid.UUID) {
	b.publishPresence(roomID, userID, "", "leave")
}

// PublishToRemotePods publishes a message to remote pods only (no local broadcast)
func (b *PGBridge) PublishToRemotePods(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("pgbridge: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

// publishRoom sends a room message via NOTIFY
func (b *PGBridge) publishRoom(roomID string, data []byte) {
	envelope := roomMessage{
		PodID:   b.podID,
		RoomID:  roomID,
		Message: data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("pgbridge: failed to marshal room envelope", "error", err)
		return
	}

	_, err = b.pool.Exec(context.Background(), "SELECT pg_notify($1, $2)", channelRoom, string(payload))
	if err != nil {
		slog.Error("pgbridge: failed to publish room message", "error", err)
	}
}

// publishPresence sends a presence message via NOTIFY
func (b *PGBridge) publishPresence(roomID string, userID uuid.UUID, userName string, action string) {
	envelope := presenceMessage{
		PodID:    b.podID,
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
		Action:   action,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("pgbridge: failed to marshal presence envelope", "error", err)
		return
	}

	_, err = b.pool.Exec(context.Background(), "SELECT pg_notify($1, $2)", channelPresence, string(payload))
	if err != nil {
		slog.Error("pgbridge: failed to publish presence message", "error", err)
	}
}

// handleRoomNotification processes incoming room messages from other pods
func (b *PGBridge) handleRoomNotification(payload string) {
	var env roomMessage
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		slog.Error("pgbridge: failed to unmarshal room notification", "error", err)
		return
	}

	// Ignore own messages
	if env.PodID == b.podID {
		return
	}

	slog.Debug("pgbridge: received room message from other pod",
		"fromPod", env.PodID,
		"roomId", env.RoomID,
	)

	// Inject into local hub
	b.hub.BroadcastRaw(env.RoomID, env.Message)
}

// handlePresenceNotification processes incoming presence events from other pods
func (b *PGBridge) handlePresenceNotification(payload string) {
	var env presenceMessage
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		slog.Error("pgbridge: failed to unmarshal presence notification", "error", err)
		return
	}

	// Ignore own messages
	if env.PodID == b.podID {
		return
	}

	switch env.Action {
	case "join":
		b.handlePresenceJoin(env)
	case "leave":
		b.handlePresenceLeave(env)
	}
}

// handlePresenceJoin adds a remote user to the presence map
func (b *PGBridge) handlePresenceJoin(env presenceMessage) {
	slog.Debug("pgbridge: remote user joined",
		"userId", env.UserID.String(),
		"userName", env.UserName,
		"roomId", env.RoomID,
		"fromPod", env.PodID,
	)

	// Cancel any pending disconnect for this user (handles cross-pod reconnection)
	b.hub.CancelPendingDisconnect(env.RoomID, env.UserID)

	b.mu.Lock()
	if b.remoteUsers[env.RoomID] == nil {
		b.remoteUsers[env.RoomID] = make(map[string]RemoteUser)
	}
	b.remoteUsers[env.RoomID][env.UserID.String()] = RemoteUser{
		UserID:   env.UserID,
		UserName: env.UserName,
		PodID:    env.PodID,
	}
	b.mu.Unlock()
}

// handlePresenceLeave removes a remote user from the presence map
func (b *PGBridge) handlePresenceLeave(env presenceMessage) {
	slog.Debug("pgbridge: remote user left",
		"userId", env.UserID.String(),
		"roomId", env.RoomID,
		"fromPod", env.PodID,
	)

	b.mu.Lock()
	if room, exists := b.remoteUsers[env.RoomID]; exists {
		delete(room, env.UserID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, env.RoomID)
		}
	}
	b.mu.Unlock()
}
