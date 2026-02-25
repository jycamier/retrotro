# MessageBus Interface Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract a `MessageBus` interface from the concrete `PGBridge`, migrate the PostgreSQL implementation, and add a NATS implementation, selectable via `BUS_TYPE` env var.

**Architecture:** New package `backend/internal/bus/` defines the `MessageBus` interface. Two implementations (`PGBus`, `NATSBus`) live in the same package. A factory reads `BUS_TYPE` from config and returns the appropriate implementation via fx. All consumers (`websocket_handler`, `timer_service`) depend on the interface, not the concrete type.

**Tech Stack:** Go 1.24, uber/fx, pgx/v5 (PG impl), nats.go (NATS impl)

---

### Task 1: Create the MessageBus interface

**Files:**
- Create: `backend/internal/bus/bus.go`

**Step 1: Write the interface file**

```go
package bus

import (
	"context"

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
```

**Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success (no consumers yet)

**Step 3: Commit**

```bash
git add backend/internal/bus/bus.go
git commit -m "feat(bus): add MessageBus interface"
```

---

### Task 2: Add BusType and NatsURL to config

**Files:**
- Modify: `backend/internal/config/config.go`

**Step 1: Add fields to Config struct**

Add after the `JWT` field in the `Config` struct:

```go
BusType string
NatsURL string
```

**Step 2: Read env vars in Load()**

Add in the `Load()` function return, after `JWT`:

```go
BusType: getEnv("BUS_TYPE", "postgres"),
NatsURL: getEnv("NATS_URL", ""),
```

**Step 3: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add backend/internal/config/config.go
git commit -m "feat(config): add BUS_TYPE and NATS_URL env vars"
```

---

### Task 3: Migrate PGBridge to PGBus

Move the existing `pgbridge/bridge.go` code into `bus/pgbus.go`, renaming the struct to `PGBus` and ensuring it satisfies `MessageBus`.

**Files:**
- Create: `backend/internal/bus/pgbus.go`
- Reference: `backend/internal/pgbridge/bridge.go` (copy and rename)

**Step 1: Create pgbus.go**

Copy `backend/internal/pgbridge/bridge.go` into `backend/internal/bus/pgbus.go` with these changes:
- Package: `bus` (not `pgbridge`)
- Rename `PGBridge` → `PGBus`
- Rename `NewPGBridge` → `NewPGBus`
- Keep all method signatures identical (they already match the interface)

**Step 2: Add compile-time interface check**

At the top of `pgbus.go`, after imports:

```go
var _ MessageBus = (*PGBus)(nil)
```

**Step 3: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 4: Commit**

```bash
git add backend/internal/bus/pgbus.go
git commit -m "feat(bus): migrate PGBridge to PGBus implementing MessageBus"
```

---

### Task 4: Create NATSBus implementation

**Files:**
- Create: `backend/internal/bus/natsbus.go`

**Step 1: Add nats.go dependency**

Run: `cd backend && go get github.com/nats-io/nats.go`

**Step 2: Write natsbus.go**

The NATS implementation follows the same pattern as PGBus: local broadcast via Hub, relay to other pods via NATS subjects. Two subjects: `retrotro.room` and `retrotro.presence`.

```go
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

var _ MessageBus = (*NATSBus)(nil)

const (
	natsSubjectRoom     = "retrotro.room"
	natsSubjectPresence = "retrotro.presence"
)

type NATSBus struct {
	hub         *websocket.Hub
	nc          *nats.Conn
	podID       string
	mu          sync.RWMutex
	remoteUsers map[string]map[string]RemoteUser
	subs        []*nats.Subscription
}

func NewNATSBus(hub *websocket.Hub, nc *nats.Conn) *NATSBus {
	return &NATSBus{
		hub:         hub,
		nc:          nc,
		podID:       uuid.New().String(),
		remoteUsers: make(map[string]map[string]RemoteUser),
	}
}

func (b *NATSBus) Start(_ context.Context) error {
	roomSub, err := b.nc.Subscribe(natsSubjectRoom, func(msg *nats.Msg) {
		b.handleRoomNotification(string(msg.Data))
	})
	if err != nil {
		return err
	}

	presenceSub, err := b.nc.Subscribe(natsSubjectPresence, func(msg *nats.Msg) {
		b.handlePresenceNotification(string(msg.Data))
	})
	if err != nil {
		roomSub.Unsubscribe()
		return err
	}

	b.subs = []*nats.Subscription{roomSub, presenceSub}
	slog.Info("natsbus: listening",
		"podId", b.podID,
		"subjects", []string{natsSubjectRoom, natsSubjectPresence},
	)
	return nil
}

func (b *NATSBus) Stop() {
	for _, sub := range b.subs {
		sub.Unsubscribe()
	}
	// Do not close the NATS connection here — it is managed externally
}

func (b *NATSBus) Hub() *websocket.Hub {
	return b.hub
}

func (b *NATSBus) BroadcastToRoom(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("natsbus: failed to marshal message", "error", err)
		return
	}
	b.hub.BroadcastRaw(roomID, data)
	b.publishRoom(roomID, data)
}

func (b *NATSBus) BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client) {
	b.hub.BroadcastToRoomExcept(roomID, msg, exclude)
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("natsbus: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

func (b *NATSBus) PublishToRemotePods(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("natsbus: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

func (b *NATSBus) GetRoomClients(roomID string) []*websocket.Client {
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

func (b *NATSBus) IsUserInRoom(roomID string, userID uuid.UUID) bool {
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

func (b *NATSBus) PublishPresenceJoin(roomID string, userID uuid.UUID, userName string) {
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

func (b *NATSBus) PublishPresenceLeave(roomID string, userID uuid.UUID) {
	b.publishPresence(roomID, userID, "", "leave")
}

// --- private ---

func (b *NATSBus) publishRoom(roomID string, data []byte) {
	envelope := roomMessage{
		PodID:   b.podID,
		RoomID:  roomID,
		Message: data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("natsbus: failed to marshal room envelope", "error", err)
		return
	}
	if err := b.nc.Publish(natsSubjectRoom, payload); err != nil {
		slog.Error("natsbus: failed to publish room message", "error", err)
	}
}

func (b *NATSBus) publishPresence(roomID string, userID uuid.UUID, userName string, action string) {
	envelope := presenceMessage{
		PodID:    b.podID,
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
		Action:   action,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("natsbus: failed to marshal presence envelope", "error", err)
		return
	}
	if err := b.nc.Publish(natsSubjectPresence, payload); err != nil {
		slog.Error("natsbus: failed to publish presence message", "error", err)
	}
}

func (b *NATSBus) handleRoomNotification(payload string) {
	var env roomMessage
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		slog.Error("natsbus: failed to unmarshal room notification", "error", err)
		return
	}
	if env.PodID == b.podID {
		return
	}
	b.hub.BroadcastRaw(env.RoomID, env.Message)
}

func (b *NATSBus) handlePresenceNotification(payload string) {
	var env presenceMessage
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		slog.Error("natsbus: failed to unmarshal presence notification", "error", err)
		return
	}
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

func (b *NATSBus) handlePresenceJoin(env presenceMessage) {
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

func (b *NATSBus) handlePresenceLeave(env presenceMessage) {
	b.mu.Lock()
	if room, exists := b.remoteUsers[env.RoomID]; exists {
		delete(room, env.UserID.String())
		if len(room) == 0 {
			delete(b.remoteUsers, env.RoomID)
		}
	}
	b.mu.Unlock()
}
```

**Step 3: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 4: Commit**

```bash
git add backend/internal/bus/natsbus.go backend/go.mod backend/go.sum
git commit -m "feat(bus): add NATSBus implementation"
```

---

### Task 5: Move shared types from pgbus.go to bus.go

The `roomMessage`, `presenceMessage`, and `RemoteUser` types are used by both implementations. Move them to `bus.go`.

**Files:**
- Modify: `backend/internal/bus/bus.go` (add shared types)
- Modify: `backend/internal/bus/pgbus.go` (remove type declarations already moved)

**Step 1: Add shared types to bus.go**

Append after the interface:

```go
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
```

Add `"encoding/json"` to bus.go imports.

**Step 2: Remove these type declarations from pgbus.go**

Remove `RemoteUser`, `roomMessage`, `presenceMessage` struct definitions from pgbus.go (they now live in bus.go).

**Step 3: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 4: Commit**

```bash
git add backend/internal/bus/
git commit -m "refactor(bus): extract shared types to bus.go"
```

---

### Task 6: Create factory and fx module

**Files:**
- Create: `backend/internal/bus/factory.go`

**Step 1: Write factory.go**

```go
package bus

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var Module = fx.Module("bus",
	fx.Provide(NewMessageBusFx),
)

func NewMessageBusFx(lc fx.Lifecycle, hub *websocket.Hub, pool *pgxpool.Pool, cfg *config.Config) (MessageBus, error) {
	switch cfg.BusType {
	case "nats":
		nc, err := nats.Connect(cfg.NatsURL)
		if err != nil {
			return nil, fmt.Errorf("bus: failed to connect to NATS at %s: %w", cfg.NatsURL, err)
		}

		b := NewNATSBus(hub, nc)
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if err := b.Start(ctx); err != nil {
					slog.Error("natsbus: failed to start", "error", err)
					return err
				}
				slog.Info("natsbus: started", "podId", b.podID, "url", cfg.NatsURL)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				b.Stop()
				nc.Close()
				slog.Info("natsbus: stopped")
				return nil
			},
		})
		return b, nil

	case "postgres", "":
		b := NewPGBus(hub, pool)
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if err := b.Start(ctx); err != nil {
					slog.Error("pgbus: failed to start", "error", err)
					return err
				}
				slog.Info("pgbus: started", "podId", b.podID)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				b.Stop()
				slog.Info("pgbus: stopped")
				return nil
			},
		})
		return b, nil

	default:
		return nil, fmt.Errorf("bus: unknown BUS_TYPE %q (expected \"postgres\" or \"nats\")", cfg.BusType)
	}
}
```

**Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 3: Commit**

```bash
git add backend/internal/bus/factory.go
git commit -m "feat(bus): add factory with fx lifecycle for PG/NATS selection"
```

---

### Task 7: Update consumers to use MessageBus interface

**Files:**
- Modify: `backend/internal/handlers/websocket_handler.go` (lines 15, 32, 53-54)
- Modify: `backend/internal/handlers/fx.go` (lines 7, 36-46)
- Modify: `backend/internal/services/timer_service.go` (lines 8, 44, 52)
- Modify: `backend/internal/services/fx.go` (lines 8, 49)

**Step 1: Update websocket_handler.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Change struct field:
```go
bridge         *pgbridge.PGBridge
```
to:
```go
bridge         bus.MessageBus
```

Change constructor parameter:
```go
bridge *pgbridge.PGBridge,
```
to:
```go
bridge bus.MessageBus,
```

**Step 2: Update handlers/fx.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Change `NewWebSocketHandlerFx` parameter:
```go
bridge *pgbridge.PGBridge,
```
to:
```go
bridge bus.MessageBus,
```

**Step 3: Update timer_service.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Change struct field and constructor parameter from `*pgbridge.PGBridge` to `bus.MessageBus`.

**Step 4: Update services/fx.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Change `NewTimerServiceFx` parameter from `*pgbridge.PGBridge` to `bus.MessageBus`.

**Step 5: Update cmd/server/main.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Replace `pgbridge.Module` with `bus.Module`.

**Step 6: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: success

**Step 7: Commit**

```bash
git add backend/internal/handlers/ backend/internal/services/ backend/cmd/server/main.go
git commit -m "refactor: consumers depend on bus.MessageBus interface"
```

---

### Task 8: Delete old pgbridge package

**Files:**
- Delete: `backend/internal/pgbridge/bridge.go`
- Delete: `backend/internal/pgbridge/fx.go`

**Step 1: Remove the directory**

```bash
rm -r backend/internal/pgbridge/
```

**Step 2: Verify no remaining references**

Run: `cd backend && grep -r "pgbridge" --include="*.go" .`
Expected: no matches

**Step 3: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add -A backend/internal/pgbridge/
git commit -m "refactor: remove old pgbridge package"
```

---

### Task 9: Run E2E tests

The interface change is transparent to the frontend. Default `BUS_TYPE` is `postgres`, so existing Docker Compose setup works unchanged.

**Step 1: Run E2E tests**

Run: `cd tests && npx playwright test`
Expected: all tests pass (10 passed, 1 skipped)

**Step 2: Final commit if any fix needed**

If tests pass with no changes, no commit needed.
