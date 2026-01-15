package leaderboardintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

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

	// Reset environment for clean state
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}


	realDB := &leaderboarddb.LeaderboardDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &leaderboardmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_leaderboard_service")

	// Create the LeaderboardService with no-op dependencies
	// NewLeaderboardService signature: NewLeaderboardService(db *bun.DB, repo leaderboarddb.LeaderboardDB, logger *slog.Logger, metrics leaderboardmetrics.LeaderboardMetrics, tracer trace.Tracer)
	service := leaderboardservice.NewLeaderboardService(
		env.DB,
		realDB,
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
