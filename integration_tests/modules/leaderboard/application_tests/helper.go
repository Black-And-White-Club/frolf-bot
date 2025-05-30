package leaderboardintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// Global variables for the test environment, initialized once.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

type TestDeps struct {
	Ctx     context.Context
	DB      leaderboarddb.LeaderboardDB
	BunDB   *bun.DB
	Service leaderboardservice.Service
	Cleanup func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing leaderboard test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Leaderboard test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Leaderboard test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Leaderboard test environment not initialized")
	}

	return testEnv
}

func SetupTestLeaderboardService(t *testing.T) TestDeps {
	t.Helper()

	// Get the shared test environment
	env := GetTestEnv(t)

	// Check if containers should be recreated for stability
	if err := env.MaybeRecreateContainers(context.Background()); err != nil {
		t.Fatalf("Failed to handle container recreation: %v", err)
	}

	// Perform deep cleanup between tests for better isolation
	if err := env.DeepCleanup(); err != nil {
		t.Fatalf("Failed to perform deep cleanup: %v", err)
	}

	// Clean up database tables before each test
	if err := testutils.TruncateTables(env.Ctx, env.DB, "leaderboards"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	realDB := &leaderboarddb.LeaderboardDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &leaderboardmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_leaderboard_service")

	// Create the LeaderboardService with no-op dependencies
	service := leaderboardservice.NewLeaderboardService(
		realDB,
		nil, // No EventBus needed for leaderboard service
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	cleanup := func() {
		// No special cleanup needed for leaderboard service
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

func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}

func boolPtr(b bool) *bool {
	return &b
}
