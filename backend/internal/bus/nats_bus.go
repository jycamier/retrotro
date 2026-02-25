package bus

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

// Compile-time check that NATSDirectBus implements MessageBus.
var _ MessageBus = (*NATSDirectBus)(nil)

// natsEnvelope wraps a WS message with the sender pod ID so we can ignore our own messages.
type natsEnvelope struct {
	PodID   string          `json:"podId"`
	Message json.RawMessage `json:"message"`
}

// natsPresenceMessage is published on presence subjects.
type natsPresenceMessage struct {
	PodID    string    `json:"podId"`
	UserID   uuid.UUID `json:"userId"`
	UserName string    `json:"userName,omitempty"`
}

// NATSDirectBus implements MessageBus using native NATS connections (no Watermill).
// This was proven to work in the feat/nats POC.
type NATSDirectBus struct {
	hub         *websocket.Hub
	conn        *nats.Conn
	podID       string
	mu          sync.RWMutex
	remoteUsers map[string]map[string]RemoteUser // roomID -> userID -> RemoteUser
	subs        []*nats.Subscription
}

// NewNATSDirectBus creates a new bus backed by a native NATS connection.
func NewNATSDirectBus(hub *websocket.Hub, conn *nats.Conn) *NATSDirectBus {
	return &NATSDirectBus{
		hub:         hub,
		conn:        conn,
		podID:       uuid.New().String(),
		remoteUsers: make(map[string]map[string]RemoteUser),
	}
}

// Hub returns the underlying websocket.Hub.
func (b *NATSDirectBus) Hub() *websocket.Hub {
	return b.hub
}

// Start subscribes to NATS subjects for room broadcasts and presence.
func (b *NATSDirectBus) Start(_ context.Context) error {
	sub, err := b.conn.Subscribe("retrotro.room.*", b.handleRoomMessage)
	if err != nil {
		return err
	}
	b.subs = append(b.subs, sub)

	sub, err = b.conn.Subscribe("retrotro.presence.join.*", b.handlePresenceJoin)
	if err != nil {
		return err
	}
	b.subs = append(b.subs, sub)

	sub, err = b.conn.Subscribe("retrotro.presence.leave.*", b.handlePresenceLeave)
	if err != nil {
		return err
	}
	b.subs = append(b.subs, sub)

	slog.Info("nats direct bus: subscribed", "podId", b.podID)
	return nil
}

// Stop unsubscribes and drains the NATS connection.
func (b *NATSDirectBus) Stop() {
	for _, sub := range b.subs {
		_ = sub.Unsubscribe()
	}
	b.subs = nil
	if b.conn != nil {
		b.conn.Close()
	}
}

// BroadcastToRoom broadcasts locally and publishes to NATS.
func (b *NATSDirectBus) BroadcastToRoom(roomID string, msg websocket.Message) {
	b.hub.BroadcastToRoom(roomID, msg)
	b.publishToNATS(roomID, msg)
}

// BroadcastToRoomExcept broadcasts locally with exclude and publishes to NATS.
func (b *NATSDirectBus) BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client) {
	b.hub.BroadcastToRoomExcept(roomID, msg, exclude)
	b.publishToNATS(roomID, msg)
}

// PublishToRemotePods sends a message only to remote pods.
func (b *NATSDirectBus) PublishToRemotePods(roomID string, msg websocket.Message) {
	b.publishToNATS(roomID, msg)
}

// GetRoomClients returns local clients merged with remote users.
func (b *NATSDirectBus) GetRoomClients(roomID string) []*websocket.Client {
	locals := b.hub.GetRoomClients(roomID)

	localUserIDs := make(map[uuid.UUID]bool, len(locals))
	for _, c := range locals {
		localUserIDs[c.UserID] = true
	}

	b.mu.RLock()
	remotes := b.remoteUsers[roomID]
	b.mu.RUnlock()

	for _, ru := range remotes {
		if !localUserIDs[ru.UserID] {
			locals = append(locals, &websocket.Client{
				ID:       "remote-" + ru.UserID.String(),
				UserID:   ru.UserID,
				UserName: ru.UserName,
				RoomID:   roomID,
			})
		}
	}

	return locals
}

// IsUserInRoom checks locally and in remote users.
func (b *NATSDirectBus) IsUserInRoom(roomID string, userID uuid.UUID) bool {
	if b.hub.IsUserInRoom(roomID, userID) {
		return true
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if room, ok := b.remoteUsers[roomID]; ok {
		_, exists := room[userID.String()]
		return exists
	}
	return false
}

// PublishPresenceJoin publishes a presence join event to NATS.
func (b *NATSDirectBus) PublishPresenceJoin(roomID string, userID uuid.UUID, userName string) {
	b.mu.Lock()
	if room, ok := b.remoteUsers[roomID]; ok {
		delete(room, userID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, roomID)
		}
	}
	b.mu.Unlock()

	msg := natsPresenceMessage{
		PodID:    b.podID,
		UserID:   userID,
		UserName: userName,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("nats: failed to marshal presence join", "error", err)
		return
	}
	if err := b.conn.Publish("retrotro.presence.join."+roomID, data); err != nil {
		slog.Error("nats: failed to publish presence join", "error", err)
	}
}

// PublishPresenceLeave publishes a presence leave event to NATS.
func (b *NATSDirectBus) PublishPresenceLeave(roomID string, userID uuid.UUID) {
	msg := natsPresenceMessage{
		PodID:  b.podID,
		UserID: userID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("nats: failed to marshal presence leave", "error", err)
		return
	}
	if err := b.conn.Publish("retrotro.presence.leave."+roomID, data); err != nil {
		slog.Error("nats: failed to publish presence leave", "error", err)
	}
}

// --- internal ---

func (b *NATSDirectBus) publishToNATS(roomID string, msg websocket.Message) {
	msgData, err := json.Marshal(msg)
	if err != nil {
		slog.Error("nats: failed to marshal message", "error", err)
		return
	}

	env := natsEnvelope{
		PodID:   b.podID,
		Message: msgData,
	}
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("nats: failed to marshal envelope", "error", err)
		return
	}

	if err := b.conn.Publish("retrotro.room."+roomID, data); err != nil {
		slog.Error("nats: failed to publish room message", "error", err, "roomId", roomID)
	}
}

func (b *NATSDirectBus) handleRoomMessage(msg *nats.Msg) {
	var env natsEnvelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		slog.Error("nats: failed to unmarshal room envelope", "error", err)
		return
	}

	if env.PodID == b.podID {
		return
	}

	// Extract roomID from subject: retrotro.room.<roomID>
	roomID := msg.Subject[len("retrotro.room."):]

	slog.Debug("nats: received room message from other pod",
		"fromPod", env.PodID,
		"roomId", roomID,
	)

	b.hub.BroadcastRaw(roomID, env.Message)
}

func (b *NATSDirectBus) handlePresenceJoin(msg *nats.Msg) {
	var pm natsPresenceMessage
	if err := json.Unmarshal(msg.Data, &pm); err != nil {
		slog.Error("nats: failed to unmarshal presence join", "error", err)
		return
	}

	if pm.PodID == b.podID {
		return
	}

	roomID := msg.Subject[len("retrotro.presence.join."):]

	slog.Debug("nats: remote user joined",
		"userId", pm.UserID.String(),
		"userName", pm.UserName,
		"roomId", roomID,
		"fromPod", pm.PodID,
	)

	b.hub.CancelPendingDisconnect(roomID, pm.UserID)

	b.mu.Lock()
	if b.remoteUsers[roomID] == nil {
		b.remoteUsers[roomID] = make(map[string]RemoteUser)
	}
	b.remoteUsers[roomID][pm.UserID.String()] = RemoteUser{
		UserID:   pm.UserID,
		UserName: pm.UserName,
		PodID:    pm.PodID,
	}
	b.mu.Unlock()
}

func (b *NATSDirectBus) handlePresenceLeave(msg *nats.Msg) {
	var pm natsPresenceMessage
	if err := json.Unmarshal(msg.Data, &pm); err != nil {
		slog.Error("nats: failed to unmarshal presence leave", "error", err)
		return
	}

	if pm.PodID == b.podID {
		return
	}

	roomID := msg.Subject[len("retrotro.presence.leave."):]

	slog.Debug("nats: remote user left",
		"userId", pm.UserID.String(),
		"roomId", roomID,
		"fromPod", pm.PodID,
	)

	b.mu.Lock()
	if room, ok := b.remoteUsers[roomID]; ok {
		delete(room, pm.UserID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, roomID)
		}
	}
	b.mu.Unlock()
}
