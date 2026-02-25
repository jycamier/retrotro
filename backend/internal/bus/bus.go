package bus

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

// MessageBus abstracts inter-pod messaging (broadcast + presence).
type MessageBus interface {
	BroadcastToRoom(roomID string, msg websocket.Message)
	BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client)
	GetRoomClients(roomID string) []*websocket.Client
	IsUserInRoom(roomID string, userID uuid.UUID) bool
	PublishPresenceJoin(roomID string, userID uuid.UUID, userName string)
	PublishPresenceLeave(roomID string, userID uuid.UUID)
	PublishToRemotePods(roomID string, msg websocket.Message)
	Hub() *websocket.Hub
	Start(ctx context.Context) error
	Stop()
}

// RemoteUser represents a user connected on another pod.
type RemoteUser struct {
	UserID   uuid.UUID
	UserName string
	PodID    string
}

// roomMessage is the envelope for room broadcasts between pods.
type roomMessage struct {
	PodID   string          `json:"podId"`
	RoomID  string          `json:"roomId"`
	Message json.RawMessage `json:"message"`
}

// presenceMessage is the envelope for presence events between pods.
type presenceMessage struct {
	PodID    string    `json:"podId"`
	RoomID   string    `json:"roomId"`
	UserID   uuid.UUID `json:"userId"`
	UserName string    `json:"userName,omitempty"`
	Action   string    `json:"action"`
}
