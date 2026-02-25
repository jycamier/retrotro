# MessageBus Interface Implementation Plan (Watermill)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract a `MessageBus` interface from the concrete `PGBridge` and implement it using Watermill as pub/sub transport, with backends selectable via `BUS_TYPE` env var (`gochannel`, `nats`, `sql`).

**Architecture:** New package `backend/internal/bus/` defines the `MessageBus` interface. A single struct `WatermillBus` implements it, wrapping a Watermill `message.Publisher` + `message.Subscriber` for inter-pod relay, and the local `websocket.Hub` for in-process broadcast. A factory reads `BUS_TYPE` and instantiates the appropriate Watermill adapter. All consumers depend on the interface.

**Tech Stack:** Go 1.24, uber/fx, Watermill v1.5+ (core + watermill-nats for NATS JetStream, watermill-sql for PostgreSQL polling, gochannel for dev)

---

### Task 1: Create the MessageBus interface and shared types

**Files:**
- Create: `backend/internal/bus/bus.go`

**Step 1: Write the interface file**

```go
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
```

**Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 3: Commit**

```bash
git add backend/internal/bus/bus.go
git commit -m "feat(bus): add MessageBus interface and shared types"
```

---

### Task 2: Add BusType and NatsURL to config

**Files:**
- Modify: `backend/internal/config/config.go`

**Step 1: Add fields to Config struct**

Add after the `JWT` field:

```go
BusType string
NatsURL string
```

**Step 2: Read env vars in Load()**

Add in the return struct, after `JWT`:

```go
BusType: getEnv("BUS_TYPE", "gochannel"),
NatsURL: getEnv("NATS_URL", ""),
```

Default is `gochannel` (in-memory, single pod, zero config for dev).

**Step 3: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add backend/internal/config/config.go
git commit -m "feat(config): add BUS_TYPE and NATS_URL env vars"
```

---

### Task 3: Add Watermill dependencies

**Step 1: Install Watermill core + adapters**

```bash
cd backend
go get github.com/ThreeDotsLabs/watermill@latest
go get github.com/ThreeDotsLabs/watermill-nats/v2@latest
go get github.com/ThreeDotsLabs/watermill-sql/v3@latest
```

**Step 2: Verify go.mod updated**

Run: `cd backend && grep -i watermill go.mod`
Expected: three watermill entries

**Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "feat(deps): add watermill core, nats and sql adapters"
```

---

### Task 4: Implement WatermillBus

Single implementation that wraps Watermill Publisher/Subscriber. The pub/sub transport handles inter-pod relay; the local Hub handles in-process WebSocket broadcast. Two topics: `retrotro.room` and `retrotro.presence`.

**Files:**
- Create: `backend/internal/bus/watermill_bus.go`

**Step 1: Write watermill_bus.go**

```go
package bus

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var _ MessageBus = (*WatermillBus)(nil)

const (
	topicRoom     = "retrotro.room"
	topicPresence = "retrotro.presence"
)

// WatermillBus implements MessageBus using Watermill Publisher/Subscriber.
type WatermillBus struct {
	hub         *websocket.Hub
	publisher   message.Publisher
	subscriber  message.Subscriber
	podID       string
	mu          sync.RWMutex
	remoteUsers map[string]map[string]RemoteUser
	cancel      context.CancelFunc
}

// NewWatermillBus creates a new WatermillBus.
func NewWatermillBus(hub *websocket.Hub, pub message.Publisher, sub message.Subscriber) *WatermillBus {
	return &WatermillBus{
		hub:         hub,
		publisher:   pub,
		subscriber:  sub,
		podID:       uuid.New().String(),
		remoteUsers: make(map[string]map[string]RemoteUser),
	}
}

func (b *WatermillBus) Start(ctx context.Context) error {
	listenCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	roomCh, err := b.subscriber.Subscribe(listenCtx, topicRoom)
	if err != nil {
		cancel()
		return err
	}

	presenceCh, err := b.subscriber.Subscribe(listenCtx, topicPresence)
	if err != nil {
		cancel()
		return err
	}

	go b.listenLoop(listenCtx, roomCh, b.handleRoomMessage)
	go b.listenLoop(listenCtx, presenceCh, b.handlePresenceMessage)

	slog.Info("bus: listening",
		"podId", b.podID,
		"topics", []string{topicRoom, topicPresence},
	)
	return nil
}

func (b *WatermillBus) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.publisher.Close()
	b.subscriber.Close()
}

func (b *WatermillBus) Hub() *websocket.Hub {
	return b.hub
}

// --- Broadcast ---

func (b *WatermillBus) BroadcastToRoom(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("bus: failed to marshal message", "error", err)
		return
	}
	b.hub.BroadcastRaw(roomID, data)
	b.publishRoom(roomID, data)
}

func (b *WatermillBus) BroadcastToRoomExcept(roomID string, msg websocket.Message, exclude *websocket.Client) {
	b.hub.BroadcastToRoomExcept(roomID, msg, exclude)
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("bus: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

func (b *WatermillBus) PublishToRemotePods(roomID string, msg websocket.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("bus: failed to marshal message", "error", err)
		return
	}
	b.publishRoom(roomID, data)
}

// --- Presence ---

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

func (b *WatermillBus) IsUserInRoom(roomID string, userID uuid.UUID) bool {
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

func (b *WatermillBus) PublishPresenceJoin(roomID string, userID uuid.UUID, userName string) {
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

func (b *WatermillBus) PublishPresenceLeave(roomID string, userID uuid.UUID) {
	b.publishPresence(roomID, userID, "", "leave")
}

// --- internal publish ---

func (b *WatermillBus) publishRoom(roomID string, data []byte) {
	envelope := roomMessage{
		PodID:   b.podID,
		RoomID:  roomID,
		Message: data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("bus: failed to marshal room envelope", "error", err)
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	if err := b.publisher.Publish(topicRoom, msg); err != nil {
		slog.Error("bus: failed to publish room message", "error", err)
	}
}

func (b *WatermillBus) publishPresence(roomID string, userID uuid.UUID, userName string, action string) {
	envelope := presenceMessage{
		PodID:    b.podID,
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
		Action:   action,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("bus: failed to marshal presence envelope", "error", err)
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	if err := b.publisher.Publish(topicPresence, msg); err != nil {
		slog.Error("bus: failed to publish presence message", "error", err)
	}
}

// --- internal subscribe ---

func (b *WatermillBus) listenLoop(ctx context.Context, ch <-chan *message.Message, handler func([]byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			handler(msg.Payload)
			msg.Ack()
		}
	}
}

func (b *WatermillBus) handleRoomMessage(payload []byte) {
	var env roomMessage
	if err := json.Unmarshal(payload, &env); err != nil {
		slog.Error("bus: failed to unmarshal room notification", "error", err)
		return
	}
	if env.PodID == b.podID {
		return
	}
	slog.Debug("bus: received room message from other pod",
		"fromPod", env.PodID,
		"roomId", env.RoomID,
	)
	b.hub.BroadcastRaw(env.RoomID, env.Message)
}

func (b *WatermillBus) handlePresenceMessage(payload []byte) {
	var env presenceMessage
	if err := json.Unmarshal(payload, &env); err != nil {
		slog.Error("bus: failed to unmarshal presence notification", "error", err)
		return
	}
	if env.PodID == b.podID {
		return
	}
	switch env.Action {
	case "join":
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
	case "leave":
		b.mu.Lock()
		if room, exists := b.remoteUsers[env.RoomID]; exists {
			delete(room, env.UserID.String())
			if len(room) == 0 {
				delete(b.remoteUsers, env.RoomID)
			}
		}
		b.mu.Unlock()
	}
}
```

**Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success

**Step 3: Commit**

```bash
git add backend/internal/bus/watermill_bus.go
git commit -m "feat(bus): implement WatermillBus wrapping Publisher/Subscriber"
```

---

### Task 5: Create factory and fx module

The factory creates the appropriate Watermill Publisher/Subscriber pair based on `BUS_TYPE`, then wraps them in a `WatermillBus`.

**Files:**
- Create: `backend/internal/bus/factory.go`

**Step 1: Write factory.go**

```go
package bus

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	watermillnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var Module = fx.Module("bus",
	fx.Provide(NewMessageBusFx),
)

func NewMessageBusFx(lc fx.Lifecycle, hub *websocket.Hub, pool *pgxpool.Pool, cfg *config.Config) (MessageBus, error) {
	logger := watermill.NewSlogLogger(slog.Default())

	pub, sub, err := createPubSub(cfg, pool, logger)
	if err != nil {
		return nil, err
	}

	b := NewWatermillBus(hub, pub, sub)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := b.Start(ctx); err != nil {
				slog.Error("bus: failed to start", "error", err)
				return err
			}
			slog.Info("bus: started", "podId", b.podID, "type", cfg.BusType)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			b.Stop()
			slog.Info("bus: stopped")
			return nil
		},
	})

	return b, nil
}

func createPubSub(cfg *config.Config, pool *pgxpool.Pool, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber, error) {
	switch cfg.BusType {
	case "gochannel", "":
		ch := gochannel.NewGoChannel(gochannel.Config{
			OutputChannelBuffer: 256,
		}, logger)
		return ch, ch, nil

	case "nats":
		if cfg.NatsURL == "" {
			return nil, nil, fmt.Errorf("bus: NATS_URL is required when BUS_TYPE=nats")
		}
		pub, err := watermillnats.NewPublisher(cfg.NatsURL, watermillnats.PublisherConfig{})
		if err != nil {
			return nil, nil, fmt.Errorf("bus: failed to create NATS publisher: %w", err)
		}
		sub, err := watermillnats.NewSubscriber(cfg.NatsURL, watermillnats.SubscriberConfig{})
		if err != nil {
			pub.Close()
			return nil, nil, fmt.Errorf("bus: failed to create NATS subscriber: %w", err)
		}
		return pub, sub, nil

	case "sql":
		db := stdlib.OpenDBFromPool(pool)
		pub, err := watermillsql.NewPublisher(db, watermillsql.PublisherConfig{
			SchemaAdapter:    watermillsql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: true,
		}, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("bus: failed to create SQL publisher: %w", err)
		}
		sub, err := watermillsql.NewSubscriber(db, watermillsql.SubscriberConfig{
			SchemaAdapter:  watermillsql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: watermillsql.DefaultPostgreSQLOffsetsAdapter{},
			PollInterval:   0, // use default (1s)
		}, logger)
		if err != nil {
			pub.Close()
			return nil, nil, fmt.Errorf("bus: failed to create SQL subscriber: %w", err)
		}
		return pub, sub, nil

	default:
		return nil, nil, fmt.Errorf("bus: unknown BUS_TYPE %q (expected \"gochannel\", \"nats\", or \"sql\")", cfg.BusType)
	}
}
```

Note: The `watermillsql` and `watermillnats` import paths and struct names may need adjustment based on the exact API of the installed versions. Verify during implementation by checking the actual module exports.

**Step 2: Verify it compiles**

Run: `cd backend && go build ./internal/bus/`
Expected: success. If import paths differ, adjust based on `go doc` output.

**Step 3: Commit**

```bash
git add backend/internal/bus/factory.go
git commit -m "feat(bus): add factory with gochannel/nats/sql backend selection"
```

---

### Task 6: Update consumers to use MessageBus interface

**Files:**
- Modify: `backend/internal/handlers/websocket_handler.go`
- Modify: `backend/internal/handlers/fx.go`
- Modify: `backend/internal/services/timer_service.go`
- Modify: `backend/internal/services/fx.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: Update websocket_handler.go**

Replace import `"github.com/jycamier/retrotro/backend/internal/pgbridge"` with `"github.com/jycamier/retrotro/backend/internal/bus"`.

Change struct field and constructor parameter:
- `bridge *pgbridge.PGBridge` → `bridge bus.MessageBus`

**Step 2: Update handlers/fx.go**

Replace import, change `NewWebSocketHandlerFx` parameter:
- `bridge *pgbridge.PGBridge` → `bridge bus.MessageBus`

**Step 3: Update timer_service.go**

Replace import, change struct field and constructor parameter:
- `bridge *pgbridge.PGBridge` → `bridge bus.MessageBus`

**Step 4: Update services/fx.go**

Replace import, change `NewTimerServiceFx` parameter:
- `bridge *pgbridge.PGBridge` → `bridge bus.MessageBus`

**Step 5: Update cmd/server/main.go**

Replace import `pgbridge` → `bus`. Replace `pgbridge.Module` → `bus.Module`.

**Step 6: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: success

**Step 7: Commit**

```bash
git add backend/internal/handlers/ backend/internal/services/ backend/cmd/server/main.go
git commit -m "refactor: consumers depend on bus.MessageBus interface"
```

---

### Task 7: Delete old pgbridge package

**Files:**
- Delete: `backend/internal/pgbridge/bridge.go`
- Delete: `backend/internal/pgbridge/fx.go`

**Step 1: Remove the directory**

```bash
rm -r backend/internal/pgbridge/
```

**Step 2: Verify no remaining references**

Run: `grep -r "pgbridge" backend/ --include="*.go"`
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

### Task 8: Run E2E tests

The interface change is transparent to the frontend. Default `BUS_TYPE=gochannel` works for single-pod E2E tests.

**Step 1: Run E2E tests**

Run: `cd tests && npx playwright test`
Expected: all tests pass (10 passed, 1 skipped)

**Step 2: Final commit if any fix needed**

If tests pass with no changes, no commit needed.
