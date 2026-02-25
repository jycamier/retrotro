package bus

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

// Module is the fx module for the message bus.
var Module = fx.Module("bus",
	fx.Provide(NewMessageBusFx),
)

// NewMessageBusFx creates a MessageBus and registers lifecycle hooks with fx.
func NewMessageBusFx(lc fx.Lifecycle, hub *websocket.Hub, pool *pgxpool.Pool, cfg *config.Config) (MessageBus, error) {
	switch cfg.BusType {
	case "nats":
		return newNATSBus(lc, hub, cfg)
	default:
		return newWatermillBus(lc, hub, pool, cfg)
	}
}

// newNATSBus creates a NATSDirectBus using native NATS connections.
func newNATSBus(lc fx.Lifecycle, hub *websocket.Hub, cfg *config.Config) (MessageBus, error) {
	if cfg.NatsURL == "" {
		return nil, fmt.Errorf("bus: BusType is \"nats\" but NatsURL is empty")
	}

	slog.Info("bus: connecting to NATS (direct)", "url", cfg.NatsURL)

	var natsOpts []nats.Option
	if cfg.NatsCredentials != "" {
		natsOpts = append(natsOpts, nats.UserCredentials(cfg.NatsCredentials))
	}

	conn, err := nats.Connect(cfg.NatsURL, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("bus: connect to NATS: %w", err)
	}

	bus := NewNATSDirectBus(hub, conn)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return bus.Start(ctx)
		},
		OnStop: func(_ context.Context) error {
			bus.Stop()
			return nil
		},
	})

	slog.Info("bus: NATS direct bus created successfully")
	return bus, nil
}

// newWatermillBus creates a WatermillBus for gochannel or sql backends.
func newWatermillBus(lc fx.Lifecycle, hub *websocket.Hub, pool *pgxpool.Pool, cfg *config.Config) (MessageBus, error) {
	logger := watermill.NewSlogLogger(slog.Default())

	pub, sub, err := createPubSub(cfg, pool, logger)
	if err != nil {
		return nil, fmt.Errorf("bus: create pub/sub: %w", err)
	}

	bus := NewWatermillBus(hub, pub, sub)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return bus.Start(ctx)
		},
		OnStop: func(_ context.Context) error {
			bus.Stop()
			return nil
		},
	})

	return bus, nil
}

// createPubSub builds the Watermill Publisher and Subscriber for non-NATS backends.
func createPubSub(cfg *config.Config, pool *pgxpool.Pool, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber, error) {
	switch cfg.BusType {
	case "gochannel", "":
		ch := gochannel.NewGoChannel(
			gochannel.Config{OutputChannelBuffer: 256},
			logger,
		)
		return ch, ch, nil

	case "sql":
		if pool == nil {
			return nil, nil, fmt.Errorf("bus: BusType is \"sql\" but pgxpool is nil")
		}

		db := stdlib.OpenDBFromPool(pool)

		schemaAdapter := watermillsql.DefaultPostgreSQLSchema{}
		offsetsAdapter := watermillsql.DefaultPostgreSQLOffsetsAdapter{}

		pub, err := watermillsql.NewPublisher(
			db,
			watermillsql.PublisherConfig{
				SchemaAdapter:        schemaAdapter,
				AutoInitializeSchema: true,
			},
			logger,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("bus: create sql publisher: %w", err)
		}

		sub, err := watermillsql.NewSubscriber(
			db,
			watermillsql.SubscriberConfig{
				SchemaAdapter:    schemaAdapter,
				OffsetsAdapter:   offsetsAdapter,
				InitializeSchema: true,
			},
			logger,
		)
		if err != nil {
			_ = pub.Close()
			return nil, nil, fmt.Errorf("bus: create sql subscriber: %w", err)
		}

		return pub, sub, nil

	default:
		return nil, nil, fmt.Errorf("bus: unknown BusType %q (valid: gochannel, nats, sql)", cfg.BusType)
	}
}
