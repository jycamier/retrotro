# Design : Interface MessageBus (Watermill)

## Contexte

Le PGBridge actuel (`backend/internal/pgbridge/`) couple le broadcast inter-pods a PostgreSQL LISTEN/NOTIFY. Cette approche ne fonctionne pas avec les bases serverless (Scaleway) qui utilisent PgBouncer en mode transaction pooling.

On extrait une interface `MessageBus` avec une unique implementation `WatermillBus` qui delegue le transport a Watermill (pub/sub library Go). Watermill fournit des adapters pour GoChannel (in-memory), NATS JetStream, et PostgreSQL SQL polling.

## Interface

```go
// bus/bus.go
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

## Structure du package

```
backend/internal/bus/
  bus.go             -- interface MessageBus + shared types
  watermill_bus.go   -- implementation unique wrappant Watermill Publisher/Subscriber
  factory.go         -- factory fx : lit BUS_TYPE, cree le bon adapter Watermill
```

Le package `pgbridge/` est supprime.

## Selection du backend

Variable d'environnement `BUS_TYPE` :
- `gochannel` (defaut) : in-memory, single pod, ideal pour le dev
- `nats` : NATS JetStream, multi-pod, production Scaleway. Necessite `NATS_URL`.
- `sql` : PostgreSQL SQL polling via watermill-sql, multi-pod, fonctionne partout y compris serverless

La factory fx cree le Publisher/Subscriber Watermill correspondant et les injecte dans `WatermillBus`.

## Consumers impactes

- `websocket_handler.go` : `*pgbridge.PGBridge` -> `bus.MessageBus`
- `timer_service.go` : `*pgbridge.PGBridge` -> `bus.MessageBus`
- `handlers/fx.go` : changement de type dans NewWebSocketHandlerFx
- `services/fx.go` : changement de type dans NewTimerServiceFx
- `cmd/server/main.go` : remplacer `pgbridge.Module` par `bus.Module`

## Config

Ajout dans la struct Config :
- `BusType string` (env: `BUS_TYPE`, defaut: `"gochannel"`)
- `NatsURL string` (env: `NATS_URL`, requis si BUS_TYPE=nats)

## Dependencies

- `github.com/ThreeDotsLabs/watermill` (core)
- `github.com/ThreeDotsLabs/watermill-nats/v2` (NATS JetStream adapter)
- `github.com/ThreeDotsLabs/watermill-sql/v3` (PostgreSQL SQL adapter)
