# Design : Interface MessageBus

## Contexte

Le PGBridge actuel (`backend/internal/pgbridge/`) couple le broadcast inter-pods a PostgreSQL LISTEN/NOTIFY. Cette approche ne fonctionne pas avec les bases serverless (Scaleway) qui utilisent PgBouncer en mode transaction pooling.

On extrait une interface `MessageBus` avec deux implementations : PostgreSQL (existant) et NATS (nouveau).

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
  bus.go        -- interface MessageBus
  pgbus.go      -- implementation PostgreSQL LISTEN/NOTIFY (migration du code pgbridge)
  natsbus.go    -- implementation NATS
  factory.go    -- factory fx : lit BUS_TYPE, instancie pgbus ou natsbus
```

Le package `pgbridge/` est supprime.

## Selection de l'implementation

Variable d'environnement `BUS_TYPE` :
- `postgres` (defaut) : utilise PostgreSQL LISTEN/NOTIFY
- `nats` : utilise NATS, necessite `NATS_URL`

La factory fx lit la config et retourne l'implementation appropriee.

## Consumers impactes

- `websocket_handler.go` : `*pgbridge.PGBridge` -> `bus.MessageBus`
- `timer_service.go` : `*pgbridge.PGBridge` -> `bus.MessageBus`
- `handlers/fx.go` : changement de type dans NewWebSocketHandlerFx
- `services/fx.go` : changement de type dans NewTimerServiceFx
- `cmd/server/main.go` : remplacer `pgbridge.Module` par `bus.Module`

## Config

Ajout dans la struct Config :
- `BusType string` (env: `BUS_TYPE`, defaut: `"postgres"`)
- `NatsURL string` (env: `NATS_URL`, requis si BUS_TYPE=nats)
