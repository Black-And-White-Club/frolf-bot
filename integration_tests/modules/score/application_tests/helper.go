package scoreintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
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
	DB      scoredb.ScoreDB
	BunDB   *bun.DB
	Service scoreservice.Service
	Cleanup func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing score test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Score test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Score test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Score test environment not initialized")
	}

	return testEnv
}

func SetupTestScoreService(t *testing.T) TestDeps {
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
	if err := testutils.TruncateTables(env.Ctx, env.DB, "scores"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	realDB := &scoredb.ScoreDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &scoremetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_score_service")

	// Create the ScoreService with no-op dependencies
	service := scoreservice.NewScoreService(
		realDB,
		nil, // No EventBus needed for score service
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	cleanup := func() {
		// No special cleanup needed for score service
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
