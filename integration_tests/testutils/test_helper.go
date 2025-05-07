package testutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
	"time" // Import time for NATS connection options

	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot/db/bundb"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/containers"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type TestEnvironment struct {
	Ctx           context.Context
	PgContainer   *postgres.PostgresContainer
	NatsContainer testcontainers.Container
	DB            *bun.DB
	DBService     *bundb.DBService
	NatsConn      *nats.Conn
	JetStream     jetstream.JetStream
	Config        *config.Config
	T             *testing.T
}

func NewTestEnvironment(t *testing.T) (*TestEnvironment, error) {
	ctx := context.Background()

	pgContainer, pgConnStr, err := containers.SetupPostgresContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup postgres container: %w", err)
	}

	natsContainer, natsURL, err := containers.SetupNatsContainer(ctx)
	if err != nil {
		// Terminate Postgres container if NATS setup fails
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to setup nats container: %w", err)
	}

	sqlDB, err := sql.Open("pgx", pgConnStr)
	if err != nil {
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		if natsContainer != nil {
			natsContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to open sql DB connection: %w", err)
	}

	db := bundb.BunDB(sqlDB)

	if err := runMigrations(db); err != nil {
		db.Close()
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		if natsContainer != nil {
			natsContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	dbService, err := bundb.NewTestDBService(db)
	if err != nil {
		db.Close()
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		if natsContainer != nil {
			natsContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to create DB service: %w", err)
	}

	// Connect to NATS
	natsConn, err := nats.Connect(natsURL, nats.Timeout(10*time.Second))
	if err != nil {
		db.Close()
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		if natsContainer != nil {
			natsContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		db.Close()
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		if natsContainer != nil {
			natsContainer.Terminate(ctx)
		}
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	cfg := &config.Config{
		Postgres: config.PostgresConfig{
			DSN: pgConnStr,
		},
		NATS: config.NATSConfig{ // Assuming you have a NatsConfig struct
			URL: natsURL,
		},
	}

	return &TestEnvironment{
		Ctx:           ctx,
		PgContainer:   pgContainer,
		NatsContainer: natsContainer,
		DB:            db,
		DBService:     dbService,
		NatsConn:      natsConn,
		JetStream:     js,
		Config:        cfg,
		T:             t,
	}, nil
}

// CleanNatsStreams purges messages from the specified JetStream streams.
// This is useful for cleaning up message state between test cases.
func (env *TestEnvironment) CleanNatsStreams(ctx context.Context, streamNames ...string) error {
	if env.JetStream == nil {
		return errors.New("JetStream context is not initialized in TestEnvironment")
	}

	for _, streamName := range streamNames {
		log.Printf("Purging stream: %s", streamName)
		// Use PurgeStream to remove all messages from the stream
		_, err := env.JetStream.Stream(ctx, streamName)
		if err != nil {
			// Check if the error is because the stream doesn't exist.
			// If so, it's not a failure for cleanup purposes.
			if !strings.Contains(err.Error(), "stream not found") {
				return fmt.Errorf("failed to purge stream %q: %w", streamName, err)
			}
			log.Printf("Stream %q not found, skipping purge.", streamName)
		} else {
			log.Printf("Stream %q purged successfully.", streamName)
		}
	}
	return nil
}

func (env *TestEnvironment) Cleanup() {
	log.Println("Cleaning up test environment resources...")
	// JetStream context does not need explicit closing, it uses the NATS connection
	if env.NatsConn != nil {
		env.NatsConn.Close()
	}
	if env.DB != nil {
		env.DB.Close()
	}
	if env.NatsContainer != nil {
		// Use context.Background() for termination as the original context might be cancelled
		if err := env.NatsContainer.Terminate(context.Background()); err != nil {
			log.Printf("Error terminating NATS container: %v", err)
		}
	}
	if env.PgContainer != nil {
		// Use context.Background() for termination as the original context might be cancelled
		if err := env.PgContainer.Terminate(context.Background()); err != nil {
			log.Printf("Error terminating Postgres container: %v", err)
		}
	}
	log.Println("Test environment resources cleaned up.")
}

func TruncateTables(ctx context.Context, db *bun.DB, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}

	query := "TRUNCATE TABLE "
	for i, table := range tables {
		query += fmt.Sprintf(`"%s"`, table)
		if i < len(tables)-1 {
			query += ", "
		}
	}
	query += " CASCADE"

	log.Printf("Truncating tables: %s", tables)
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to truncate tables %v: %w", tables, err)
	}
	log.Printf("Tables %s truncated successfully", tables)

	return nil
}

func CleanUserIntegrationTables(ctx context.Context, db *bun.DB) error {
	tablesToTruncate := []string{
		"users",
	}
	return TruncateTables(ctx, db, tablesToTruncate...)
}

// Add other Clean...IntegrationTables functions here as needed

func runMigrations(db *bun.DB) error {
	ctx := context.Background()

	// Initialize migrator with user migrations first (assuming it creates the migration table)
	migrator := migrate.NewMigrator(db, usermigrations.Migrations)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tables: %w", err)
	}

	// Run migrations for each module
	if err := runModuleMigrations(ctx, db, usermigrations.Migrations, "user"); err != nil {
		return err
	}
	if err := runModuleMigrations(ctx, db, roundmigrations.Migrations, "round"); err != nil {
		return err
	}
	if err := runModuleMigrations(ctx, db, scoremigrations.Migrations, "score"); err != nil {
		return err
	}
	if err := runModuleMigrations(ctx, db, leaderboardmigrations.Migrations, "leaderboard"); err != nil {
		return err
	}

	log.Println("All migrations completed successfully")
	return nil
}

func runModuleMigrations(ctx context.Context, db *bun.DB, migrations *migrate.Migrations, moduleName string) error {
	migrator := migrate.NewMigrator(db, migrations)

	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run %s migrations: %w", moduleName, err)
	}

	if group.ID == 0 {
		log.Printf("No %s migrations to run", moduleName)
	} else {
		log.Printf("Ran %s migrations group #%d", moduleName, group.ID)
	}

	return nil
}

// WaitFor repeatedly calls a check function until it returns nil or a timeout occurs.
// This is useful for waiting for asynchronous operations to complete in integration tests.
func WaitFor(timeout, interval time.Duration, check func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Check one last time before returning timeout error
			if err := check(); err == nil {
				return nil // Success on last check
			}
			return fmt.Errorf("timed out waiting: %w", ctx.Err())
		case <-ticker.C:
			if err := check(); err == nil {
				return nil // Success
			}
			// Continue waiting if check returns an error
		}
	}
}
