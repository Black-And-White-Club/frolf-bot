package guildintegrationtests

import (
	"context"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global variables for the test environment, initialized once.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// TestDeps holds dependencies needed by individual tests.
type TestDeps struct {
	Ctx     context.Context
	DB      guilddb.GuildDB
	BunDB   *bun.DB
	Service guildservice.Service
	Cleanup func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing guild test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Guild test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Guild test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Guild test environment not initialized")
	}

	return testEnv
}

func SetupTestGuildService(t *testing.T) TestDeps {
	t.Helper()

	// Get the shared test environment
	env := GetTestEnv(t)
	log.Printf("[%s] SetupTestGuildService: Starting setup", t.Name())

	// Reset environment for clean state
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}
	log.Printf("[%s] SetupTestGuildService: Environment reset complete", t.Name())

	realDB := &guilddb.GuildDBImpl{DB: env.DB}

	// Use a logger that writes to test output for debugging
	testLogger := slog.New(slog.NewTextHandler(testWriter{t: t}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	noOpMetrics := &guildmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_guild_service")
	// Create the GuildService with no-op dependencies
	service := guildservice.NewGuildService(
		realDB,
		nil, // No event bus needed for these tests
		testLogger,
		noOpMetrics,
		noOpTracer,
	)
	log.Printf("[%s] SetupTestGuildService: Service created", t.Name())

	cleanup := func() {
		log.Printf("[%s] SetupTestGuildService: Cleanup called", t.Name())
	}

	t.Cleanup(cleanup)

	return TestDeps{
		Ctx:     env.Ctx,
		DB:      realDB,
		BunDB:   env.DB,
		Service: service,
		Cleanup: cleanup,
	}
}

// testWriter wraps a testing.T to implement io.Writer for slog
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
}
