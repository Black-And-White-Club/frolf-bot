package scoreintegrationtests

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

	// Reset environment for clean state
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	realDB := scoredb.NewRepository(env.DB)

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
