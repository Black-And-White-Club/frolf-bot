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
	// Add container lifecycle management fields
	testCount      int
	recreateAfter  int
	lastRecreation time.Time
}

// NewTestEnvironment creates a new test environment with Postgres and NATS containers
func NewTestEnvironment(t *testing.T) (*TestEnvironment, error) {
	ctx, cancel := context.WithCancel(context.Background())

	env := &TestEnvironment{
		Ctx:           ctx,
		CancelContext: cancel,
		T:             t,
		recreateAfter: 20, // Recreate containers every 20 tests
	}

	if err := env.setupContainers(ctx); err != nil {
		cancel()
		return nil, err
	}

	env.lastRecreation = time.Now()
	return env, nil
}

// setupContainers initializes all containers and connections
func (env *TestEnvironment) setupContainers(ctx context.Context) error {
	pgContainer, pgConnStr, err := containers.SetupPostgresContainer(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup postgres container: %w", err)
	}
	env.PgContainer = pgContainer

	natsContainer, natsURL, err := containers.SetupNatsContainer(ctx)
	if err != nil {
		pgContainer.Terminate(ctx)
		return fmt.Errorf("failed to setup nats container: %w", err)
	}
	env.NatsContainer = natsContainer

	sqlDB, err := sql.Open("pgx", pgConnStr)
	if err != nil {
		cleanupContainers(ctx, pgContainer, natsContainer)
		return fmt.Errorf("failed to open sql DB connection: %w", err)
	}

	db := bundb.BunDB(sqlDB)
	env.DB = db

	if err := runMigrations(db); err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	dbService, err := bundb.NewTestDBService(db)
	if err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		return fmt.Errorf("failed to create DB service: %w", err)
	}
	env.DBService = dbService

	natsConn, err := nats.Connect(natsURL, nats.Timeout(10*time.Second))
	if err != nil {
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	env.NatsConn = natsConn

	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		db.Close()
		cleanupContainers(ctx, pgContainer, natsContainer)
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	env.JetStream = js

	cfg := &config.Config{
		Postgres: config.PostgresConfig{DSN: pgConnStr},
		NATS:     config.NATSConfig{URL: natsURL},
	}
	env.Config = cfg

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
		return fmt.Errorf("failed to create EventBus: %w", err)
	}
	env.EventBus = eventBus

	return nil
}

// MaybeRecreateContainers checks if containers should be recreated and does so if needed
func (env *TestEnvironment) MaybeRecreateContainers(ctx context.Context) error {
	env.testCount++

	// Recreate containers periodically for stability
	shouldRecreate := env.testCount%env.recreateAfter == 0 ||
		time.Since(env.lastRecreation) > 45*time.Minute

	if shouldRecreate {
		log.Printf("Recreating containers after %d tests or %v elapsed for stability",
			env.testCount, time.Since(env.lastRecreation))
		return env.RecreateContainers(ctx)
	}
	return nil
}

// RecreateContainers terminates existing containers and creates new ones
func (env *TestEnvironment) RecreateContainers(ctx context.Context) error {
	log.Println("Recreating containers for stability...")

	// Store references to old containers
	oldNats := env.NatsContainer
	oldPg := env.PgContainer

	// Close connections gracefully
	if env.EventBus != nil {
		if closer, ok := env.EventBus.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Error closing EventBus: %v", err)
			}
		}
		env.EventBus = nil
	}

	if env.NatsConn != nil {
		env.NatsConn.Close()
		env.NatsConn = nil
	}

	if env.DB != nil {
		env.DB.Close()
		env.DB = nil
	}

	if env.DBService != nil {
		env.DBService = nil
	}

	if env.JetStream != nil {
		env.JetStream = nil
	}

	// Terminate old containers with timeout
	terminateCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if oldNats != nil {
		if err := oldNats.Terminate(terminateCtx); err != nil {
			log.Printf("Error terminating old NATS container: %v", err)
		}
	}
	if oldPg != nil {
		if err := oldPg.Terminate(terminateCtx); err != nil {
			log.Printf("Error terminating old PostgreSQL container: %v", err)
		}
	}

	// Small delay to ensure cleanup
	time.Sleep(2 * time.Second)

	// Create new containers and connections
	if err := env.setupContainers(ctx); err != nil {
		return fmt.Errorf("failed to recreate containers: %w", err)
	}

	env.lastRecreation = time.Now()
	log.Println("Containers successfully recreated")
	return nil
}

// CheckContainerHealth verifies that containers are running and responsive
func (env *TestEnvironment) CheckContainerHealth() error {
	ctx, cancel := context.WithTimeout(env.Ctx, 10*time.Second)
	defer cancel()

	// Check NATS container
	if env.NatsContainer != nil {
		state, err := env.NatsContainer.State(ctx)
		if err != nil || !state.Running {
			return fmt.Errorf("NATS container not healthy: running=%v, err=%v", state.Running, err)
		}
	}

	// Check PostgreSQL container
	if env.PgContainer != nil {
		state, err := env.PgContainer.State(ctx)
		if err != nil || !state.Running {
			return fmt.Errorf("PostgreSQL container not healthy: running=%v, err=%v", state.Running, err)
		}
	}

	// Test database connectivity using bun.DB
	if env.DB != nil {
		var result int
		if err := env.DB.NewSelect().ColumnExpr("1").Scan(ctx, &result); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
	}

	// Test NATS connectivity
	if env.NatsConn != nil && !env.NatsConn.IsConnected() {
		return fmt.Errorf("NATS connection not healthy")
	}

	return nil
}

// DeepCleanup performs comprehensive cleanup between tests
func (env *TestEnvironment) DeepCleanup() error {
	// Clear all NATS JetStream state
	if err := env.ResetJetStreamState(env.Ctx, "round", "user", "discord", "delayed"); err != nil {
		return fmt.Errorf("failed to reset JetStream: %w", err)
	}

	// Use existing helper functions - these handle table existence checking
	if err := CleanRoundIntegrationTables(env.Ctx, env.DB); err != nil {
		log.Printf("Warning: failed to clean round tables: %v", err)
	}

	if err := CleanUserIntegrationTables(env.Ctx, env.DB); err != nil {
		log.Printf("Warning: failed to clean user tables: %v", err)
	}

	if err := CleanScoreIntegrationTables(env.Ctx, env.DB); err != nil {
		log.Printf("Warning: failed to clean score tables: %v", err)
	}

	if err := CleanLeaderboardIntegrationTables(env.Ctx, env.DB); err != nil {
		log.Printf("Warning: failed to clean leaderboard tables: %v", err)
	}

	return nil
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

	// Terminate containers with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if env.NatsContainer != nil {
		if err := env.NatsContainer.Terminate(ctx); err != nil {
			log.Printf("Error terminating NATS container: %v", err)
		} else {
			log.Println("NATS container terminated.")
		}
	}
	if env.PgContainer != nil {
		if err := env.PgContainer.Terminate(ctx); err != nil {
			log.Printf("Error terminating Postgres container: %v", err)
		} else {
			log.Println("PostgreSQL container terminated.")
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
