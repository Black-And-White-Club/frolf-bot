package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot/db/bundb"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/containers"
)

// TestEnvironment holds all resources needed for integration testing
type TestEnvironment struct {
	Ctx           context.Context
	CancelContext context.CancelFunc
	PgContainer   *postgres.PostgresContainer
	NatsContainer testcontainers.Container
	DB            *bun.DB
	DBService     *bundb.DBService
	EventBus      eventbus.EventBus
	NatsConn      *nats.Conn
	JetStream     jetstream.JetStream
	Config        *config.Config
	T             *testing.T
}

// NewTestEnvironment creates a new test environment with Postgres and NATS containers
func NewTestEnvironment(t *testing.T) (*TestEnvironment, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pgContainer, pgConnStr, err := containers.SetupPostgresContainer(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup postgres container: %w", err)
	}

	natsContainer, natsURL, err := containers.SetupNatsContainer(ctx)
	if err != nil {
		pgContainer.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("failed to setup nats container: %w", err)
	}

	sqlDB, err := sql.Open("pgx", pgConnStr)
	if err != nil {
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to open sql DB connection: %w", err)
	}

	db := bundb.BunDB(sqlDB)

	if err := runMigrations(db); err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	dbService, err := bundb.NewTestDBService(db)
	if err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to create DB service: %w", err)
	}

	natsConn, err := nats.Connect(natsURL, nats.Timeout(10*time.Second))
	if err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	cfg := &config.Config{
		Postgres: config.PostgresConfig{DSN: pgConnStr},
		NATS:     config.NATSConfig{URL: natsURL},
	}

	// Create EventBus
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventBus, err := eventbus.NewEventBus(
		ctx,
		natsURL,
		discardLogger,
		"backend",
		eventbusmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		natsConn.Close()
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		cancel()
		return nil, fmt.Errorf("failed to create EventBus: %w", err)
	}

	return &TestEnvironment{
		Ctx:           ctx,
		CancelContext: cancel,
		PgContainer:   pgContainer,
		NatsContainer: natsContainer,
		DB:            db,
		DBService:     dbService,
		EventBus:      eventBus,
		NatsConn:      natsConn,
		JetStream:     js,
		Config:        cfg,
		T:             t,
	}, nil
}

// Cleanup tears down all resources created for testing
func (env *TestEnvironment) Cleanup() {
	log.Println("Cleaning up test environment...")
	if env.CancelContext != nil {
		env.CancelContext()
		log.Println("Context cancelled.")
	}
	if env.EventBus != nil {
		if closer, ok := env.EventBus.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Error closing EventBus: %v", err)
			} else {
				log.Println("EventBus closed.")
			}
		}
	}
	if env.NatsConn != nil {
		env.NatsConn.Close()
		log.Println("NATS connection closed.")
	}
	if env.DB != nil {
		env.DB.Close()
		log.Println("DB connection closed.")
	}
	if env.NatsContainer != nil {
		if err := env.NatsContainer.Terminate(context.Background()); err != nil {
			log.Printf("Error terminating NATS container: %v", err)
		}
	}
	if env.PgContainer != nil {
		if err := env.PgContainer.Terminate(context.Background()); err != nil {
			log.Printf("Error terminating Postgres container: %v", err)
		}
	}
	log.Println("Cleanup complete.")
}

func cleanupContainers(ctx context.Context, pg *postgres.PostgresContainer, nats testcontainers.Container) {
	if pg != nil {
		pg.Terminate(ctx)
	}
	if nats != nil {
		nats.Terminate(ctx)
	}
}
