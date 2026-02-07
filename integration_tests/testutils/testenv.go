package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
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

	configureLocalDockerAutodetect()

	env := &TestEnvironment{
		Ctx:           ctx,
		CancelContext: cancel,
		T:             t,
	}

	if err := env.setupContainers(ctx); err != nil {
		cancel()
		return nil, err
	}

	return env, nil
}

// setupContainers initializes all containers and connections
func (env *TestEnvironment) setupContainers(ctx context.Context) error {
	pgContainer, natsContainer, pgConnStr, natsURL, err := globalPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire containers from pool: %w", err)
	}
	env.PgContainer = pgContainer
	env.NatsContainer = natsContainer

	sqlDB, err := sql.Open("pgx", pgConnStr)
	if err != nil {
		globalPool.Release()
		return fmt.Errorf("failed to open sql DB connection: %w", err)
	}

	db := bundb.BunDB(sqlDB)
	env.DB = db

	if err := runMigrationsWithConnStr(db, pgConnStr); err != nil {
		db.Close()
		globalPool.Release()
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	dbService, err := bundb.NewTestDBService(db)
	if err != nil {
		db.Close()
		globalPool.Release()
		return fmt.Errorf("failed to create DB service: %w", err)
	}
	env.DBService = dbService

	natsConn, err := nats.Connect(natsURL, nats.Timeout(10*time.Second))
	if err != nil {
		db.Close()
		globalPool.Release()
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	env.NatsConn = natsConn

	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		db.Close()
		globalPool.Release()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	env.JetStream = js

	cfg := &config.Config{
		Postgres: config.PostgresConfig{DSN: pgConnStr},
		NATS:     config.NATSConfig{URL: natsURL},
	}
	env.Config = cfg

	// Create EventBus
	// By default tests discard EventBus logs to avoid noisy output. For debugging hangs
	// you can enable EventBus logs by setting STREAM_EVENTBUS_LOGS=1 in the environment.
	var eventLogger *slog.Logger
	if os.Getenv("STREAM_EVENTBUS_LOGS") == "1" {
		eventLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	} else {
		eventLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	eventBus, err := eventbus.NewEventBus(
		ctx,
		natsURL,
		eventLogger,
		"backend",
		eventbusmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		natsConn.Close()
		db.Close()
		globalPool.Release()
		return fmt.Errorf("failed to create EventBus: %w", err)
	}
	env.EventBus = eventBus

	return nil
}

// Reset cleans up the environment for the next test
func (env *TestEnvironment) Reset(ctx context.Context) error {
	// Clean up database
	if err := CleanupDatabase(ctx, env.DB); err != nil {
		return fmt.Errorf("failed to cleanup database: %w", err)
	}

	// Delete all consumers first (new)
	if err := env.DeleteJetStreamConsumers(ctx, StandardStreamNames...); err != nil {
		return fmt.Errorf("failed to delete consumers: %w", err)
	}

	// Then purge messages
	if err := env.ResetJetStreamState(ctx, StandardStreamNames...); err != nil {
		return fmt.Errorf("failed to reset JetStream: %w", err)
	}

	return nil
}

// SoftReset cleans up the environment for the next test without destroying consumers
func (env *TestEnvironment) SoftReset(ctx context.Context) error {
	// Clean up database
	if err := CleanupDatabase(ctx, env.DB); err != nil {
		return fmt.Errorf("failed to cleanup database: %w", err)
	}

	// Purge messages but keep consumers
	if err := env.ResetJetStreamState(ctx, StandardStreamNames...); err != nil {
		return fmt.Errorf("failed to reset JetStream state: %w", err)
	}

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

	// Release container pool reference (don't terminate containers)
	globalPool.Release()
	log.Println("Cleanup complete.")
}

// configureLocalDockerAutodetect sets minimal Testcontainers overrides for Colima environments
// to make the integration suite plug-and-play. It only applies when:
// - DOCKER_HOST points at a Colima socket, and
// - the respective overrides are not already set by the user.
// It does not change behavior for Docker Desktop or CI environments.
func configureLocalDockerAutodetect() {
	dh := os.Getenv("DOCKER_HOST")
	dc := os.Getenv("DOCKER_CONTEXT")
	if dh == "" && dc == "" {
		return
	}
	if !(strings.Contains(dh, ".colima/") || strings.EqualFold(dc, "colima")) {
		return
	}

	if os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
		os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
	}

	// If no host override or an alias is set, prefer a concrete IP to avoid DNS issues in Colima.
	if ho := os.Getenv("TESTCONTAINERS_HOST_OVERRIDE"); ho == "" || ho == "host.lima.internal" || ho == "host.docker.internal" {
		if ip := hostIPv4ForLocal(); ip != "" {
			os.Setenv("TESTCONTAINERS_HOST_OVERRIDE", ip)
		}
	}

	if os.Getenv("TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED") == "" {
		os.Setenv("TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED", "true")
	}
}

// hostIPv4ForLocal attempts to find a stable, non-loopback IPv4 address on the host
// that is typically reachable from Colima's VM. Prefer common macOS interfaces (en0/en1),
// otherwise fall back to the first suitable non-loopback IPv4 address.
func hostIPv4ForLocal() string {
	pickFrom := []string{"en0", "en1"}
	for _, name := range pickFrom {
		if ip := ipv4ForInterface(name); ip != "" {
			return ip
		}
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		if ip := ipv4ForInterface(iface.Name); ip != "" {
			return ip
		}
	}
	return ""
}

func ipv4ForInterface(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue
		}
		return ip.String()
	}
	return ""
}
