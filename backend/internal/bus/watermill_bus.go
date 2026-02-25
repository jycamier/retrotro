package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

// Compile-time check that WatermillBus implements MessageBus.
var _ MessageBus = (*WatermillBus)(nil)

const (
	topicRoom     = "retrotro.room"
	topicPresence = "retrotro.presence"
)

// WatermillBus implements MessageBus using Watermill for cross-pod relay
// and the local websocket.Hub for in-process broadcast.
type WatermillBus struct {
	hub   *websocket.Hub
	pub   message.Publisher
	sub   message.Subscriber
	podID string

	remoteUsers map[string]map[string]RemoteUser // roomID -> userID -> RemoteUser
	mu          sync.RWMutex

	cancel context.CancelFunc
}

// NewWatermillBus creates a new WatermillBus. The podID uniquely identifies
// this process instance so that messages published by this pod are ignored
// when received back from the message broker.
func NewWatermillBus(hub *websocket.Hub, pub message.Publisher, sub message.Subscriber) *WatermillBus {
	return &WatermillBus{
		hub:         hub,
		pub:         pub,
		sub:         sub,
		podID:       watermill.NewUUID(),
		remoteUsers: make(map[string]map[string]RemoteUser),
	}
}

// Hub returns the underlying websocket.Hub.
func (b *WatermillBus) Hub() *websocket.Hub {
	return b.hub
}

// Start subscribes to both Watermill topics and spawns consumer goroutines.
// It returns an error if any subscription fails. The provided context controls
// the lifetime of the bus; callers should also call Stop() when done.
func (b *WatermillBus) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	roomMsgs, err := b.sub.Subscribe(ctx, topicRoom)
	if err != nil {
		cancel()
		return fmt.Errorf("bus: subscribe to %s: %w", topicRoom, err)
	}

	presenceMsgs, err := b.sub.Subscribe(ctx, topicPresence)
	if err != nil {
		cancel()
		return fmt.Errorf("bus: subscribe to %s: %w", topicPresence, err)
	}

	go b.consumeRoomMessages(ctx, roomMsgs)
	go b.consumePresenceMessages(ctx, presenceMsgs)

	return nil
}

// Stop cancels the internal context, closes the publisher and subscriber.
func (b *WatermillBus) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	if err := b.pub.Close(); err != nil {
		slog.Warn("bus: error closing publisher", "err", err)
	}
	if err := b.sub.Close(); err != nil {
		slog.Warn("bus: error closing subscriber", "err", err)
	}
}

// BroadcastToRoom broadcasts a message to all local clients in the room and
// relays it to remote pods via Watermill.
func (b *WatermillBus) BroadcastToRoom(roomID string, msg websocket.Message) {
	// Local broadcast.
	b.hub.BroadcastToRoom(roomID, msg)

	// Cross-pod relay.
	if err := b.publishRoomMessage(roomID, msg); err != nil {
		slog.Error("bus: failed to publish room message", "roomId", roomID, "err", err)
	}
}

// BroadcastToRoomExcept broadcasts to all local clients except one, and relays
// to remote pods via Watermill.
func (b *WatermillBus) BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client) {
	// Local broadcast (excluding the given client).
	b.hub.BroadcastToRoomExcept(roomID, msg, exclude)

	// Cross-pod relay (remote pods have no concept of the excluded client).
	if err := b.publishRoomMessage(roomID, msg); err != nil {
		slog.Error("bus: failed to publish room message (except)", "roomId", roomID, "err", err)
	}
}

// PublishToRemotePods sends a message only to remote pods (not local clients).
// Use this when the local broadcast has already been done separately.
func (b *WatermillBus) PublishToRemotePods(roomID string, msg websocket.Message) {
	if err := b.publishRoomMessage(roomID, msg); err != nil {
		slog.Error("bus: failed to publish to remote pods", "roomId", roomID, "err", err)
	}
}

// GetRoomClients returns local + remote users in a room.
func (b *WatermillBus) GetRoomClients(roomID string) []*websocket.Client {
	localClients := b.hub.GetRoomClients(roomID)

	b.mu.RLock()
	remoteRoom, exists := b.remoteUsers[roomID]
	b.mu.RUnlock()

	if !exists || len(remoteRoom) == 0 {
		return localClients
	}

	localUserIDs := make(map[uuid.UUID]bool, len(localClients))
	for _, c := range localClients {
		localUserIDs[c.UserID] = true
	}

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

// IsUserInRoom returns true if the user is connected locally or is tracked as
// a remote user in the room.
func (b *WatermillBus) IsUserInRoom(roomID string, userID uuid.UUID) bool {
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

// PublishPresenceJoin publishes a presence-join event to remote pods.
// It also removes the user from remoteUsers if they were previously tracked as remote
// (handles the case where a user reconnects to this pod after being on another).
func (b *WatermillBus) PublishPresenceJoin(roomID string, userID uuid.UUID, userName string) {
	b.mu.Lock()
	if room, exists := b.remoteUsers[roomID]; exists {
		delete(room, userID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, roomID)
		}
	}
	b.mu.Unlock()

	env := presenceMessage{
		PodID:    b.podID,
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
		Action:   "join",
	}
	if err := b.publishPresence(env); err != nil {
		slog.Error("bus: failed to publish presence join", "roomId", roomID, "userId", userID, "err", err)
	}
}

// PublishPresenceLeave publishes a presence-leave event to remote pods.
func (b *WatermillBus) PublishPresenceLeave(roomID string, userID uuid.UUID) {
	env := presenceMessage{
		PodID:  b.podID,
		RoomID: roomID,
		UserID: userID,
		Action: "leave",
	}
	if err := b.publishPresence(env); err != nil {
		slog.Error("bus: failed to publish presence leave", "roomId", roomID, "userId", userID, "err", err)
	}
}

// --- internal helpers ---

func (b *WatermillBus) publishRoomMessage(roomID string, msg websocket.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal websocket message: %w", err)
	}
	env := roomMessage{
		PodID:   b.podID,
		RoomID:  roomID,
		Message: json.RawMessage(payload),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal room envelope: %w", err)
	}
	wm := message.NewMessage(watermill.NewUUID(), data)
	slog.Info("bus: publishing room message to NATS",
		"roomId", roomID,
		"podId", b.podID,
		"msgType", msg.Type,
		"topic", topicRoom,
	)
	err = b.pub.Publish(topicRoom, wm)
	if err != nil {
		slog.Error("bus: NATS publish failed", "err", err, "roomId", roomID)
	}
	return err
}

func (b *WatermillBus) publishPresence(env presenceMessage) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal presence envelope: %w", err)
	}
	wm := message.NewMessage(watermill.NewUUID(), data)
	slog.Info("bus: publishing presence message to NATS",
		"action", env.Action,
		"roomId", env.RoomID,
		"userId", env.UserID,
		"podId", b.podID,
		"topic", topicPresence,
	)
	return b.pub.Publish(topicPresence, wm)
}

func (b *WatermillBus) consumeRoomMessages(ctx context.Context, msgs <-chan *message.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case wm, ok := <-msgs:
			if !ok {
				return
			}
			wm.Ack()

			var env roomMessage
			if err := json.Unmarshal(wm.Payload, &env); err != nil {
				slog.Warn("bus: failed to unmarshal room message", "err", err)
				continue
			}
			// Ignore messages from this pod.
			if env.PodID == b.podID {
				continue
			}
			localClients := b.hub.GetRoomClients(env.RoomID)
			slog.Info("bus: received remote room message",
				"roomId", env.RoomID,
				"podId", env.PodID,
				"localPodId", b.podID,
				"localClientsInRoom", len(localClients),
				"messageType", string(env.Message),
			)
			b.hub.BroadcastRaw(env.RoomID, env.Message)
		}
	}
}

func (b *WatermillBus) consumePresenceMessages(ctx context.Context, msgs <-chan *message.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case wm, ok := <-msgs:
			if !ok {
				return
			}
			wm.Ack()

			var env presenceMessage
			if err := json.Unmarshal(wm.Payload, &env); err != nil {
				slog.Warn("bus: failed to unmarshal presence message", "err", err)
				continue
			}
			// Ignore messages from this pod.
			if env.PodID == b.podID {
				continue
			}
			slog.Debug("bus: received remote presence message",
				"action", env.Action,
				"roomId", env.RoomID,
				"userId", env.UserID,
				"podId", env.PodID,
			)
			b.handleRemotePresence(env)
		}
	}
}

func (b *WatermillBus) handleRemotePresence(env presenceMessage) {
	switch env.Action {
	case "join":
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
		// Cancel any local pending-disconnect timer so we don't emit a spurious
		// participant_left for a user who is still alive on another pod.
		b.hub.CancelPendingDisconnect(env.RoomID, env.UserID)

	case "leave":
		b.mu.Lock()
		if room, ok := b.remoteUsers[env.RoomID]; ok {
			delete(room, env.UserID.String())
			if len(room) == 0 {
				delete(b.remoteUsers, env.RoomID)
			}
		}
		b.mu.Unlock()

	default:
		slog.Warn("bus: unknown presence action", "action", env.Action)
	}
}
