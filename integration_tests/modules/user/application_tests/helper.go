package userintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
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
	DB      userdb.UserDB
	BunDB   *bun.DB
	Service userservice.Service
	Cleanup func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing user test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("User test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("User test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("User test environment not initialized")
	}

	return testEnv
}

func SetupTestUserService(t *testing.T) TestDeps {
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
	if err := testutils.TruncateTables(env.Ctx, env.DB, "users"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	realDB := &userdb.UserDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &usermetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_user_service")

	// Create the UserService with no-op dependencies
	service := userservice.NewUserService(
		realDB,
		nil, // No EventBus needed for user service
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	cleanup := func() {
		// No special cleanup needed for user service
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

// Helper function to create a tag number pointer
func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}
