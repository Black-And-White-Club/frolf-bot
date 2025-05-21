package userhandler_integration_tests

import (
	"context" // Import fmt for error wrapping
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global variables for the test environment, initialized once (e.g., in TestMain).
// These are assumed to be set up in a TestMain function in this package's test file.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// LeaderboardHandlerTestDeps holds shared dependencies for leaderboard handler tests.
type HandlerTestDeps struct {
	*testutils.TestEnvironment
	UserModule         *user.Module
	Router             *message.Router
	EventBus           eventbus.EventBus
	ReceivedMsgs       map[string][]*message.Message
	ReceivedMsgsMutex  *sync.Mutex
	PrometheusRegistry *prometheus.Registry
	TestObservability  observability.Observability
	TestHelpers        utils.Helpers
}

// GetTestEnv creates a new test environment for the test.
// This ensures each test gets a clean environment.
// This function assumes that the global testEnv is initialized in TestMain.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	if testEnv == nil {
		// This indicates TestMain was not run or failed.
		t.Fatalf("Global test environment not initialized. Ensure TestMain is correctly set up.")
	}
	return testEnv
}

// SetupTestUserHandler sets up the environment and dependencies for user handler tests.
// It creates a new router and event bus instance for each test function's scope.
func SetupTestUserHandler(t *testing.T) HandlerTestDeps {
	t.Helper()

	// Get a fresh test environment for this test from the global instance
	env := GetTestEnv(t)

	// Set the APP_ENV to "test" for the duration of the test run
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", oldEnv)
	})

	// Clean up NATS consumers for all streams before starting the test
	// These are the streams the user handler might interact with.
	streamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	if err := env.ResetJetStreamState(env.Ctx, streamNames...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}
	log.Println("Cleaned up NATS JetStream state for user handler streams before test")

	// Truncate relevant DB tables for a clean state per test
	if err := testutils.TruncateTables(env.Ctx, env.DB, "users"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}
	log.Println("Truncated 'users' table before test")

	userDB := &userdb.UserDBImpl{DB: env.DB}
	// Use NopLogger for quieter test logs, matching leaderboard helper
	watermillLogger := watermill.NopLogger{}

	// Create contexts for the event bus and router, managed by t.Cleanup
	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

	// Create the actual EventBus implementation for this test
	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx, // Use the test-scoped context
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)), // Discard logs in tests
		"backend",                               // Use a dummy appType
		&eventbusmetrics.NoOpMetrics{},          // Use NoOpMetrics
		noop.NewTracerProvider().Tracer("test"), // Use NoOpTracer
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Explicitly create the streams needed for user handler tests if they don't exist
	for _, streamName := range streamNames {
		// Check if the stream exists before attempting to create it
		_, err := eventBusImpl.GetJetStream().Stream(env.Ctx, streamName)
		if err != nil && strings.Contains(err.Error(), "stream not found") {
			log.Printf("Stream %q not found, creating it.", streamName)
			if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
				// Ensure cleanup on failure
				eventBusImpl.Close()
				eventBusCancel()
				t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
			}
		} else if err != nil {
			// Handle other potential errors when checking for stream existence
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to check existence of NATS stream %q: %v", streamName, err)
		} else {
			log.Printf("Stream %q already exists, skipping creation.", streamName)
		}
	}

	// Create router with test-appropriate configuration
	routerConfig := message.RouterConfig{CloseTimeout: 1 * time.Second}

	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	// Use NoOpMetrics and TracerProvider for test observability
	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), // Discard logs
		},
		Registry: &observability.Registry{
			UserMetrics: &usermetrics.NoOpMetrics{},                     // Use NoOpMetrics
			Tracer:      noop.NewTracerProvider().Tracer("test"),        // Use NoOpTracer
			Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)), // Discard logs
		},
	}

	// Use real helpers but with a discard logger
	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Create the user module
	userModule, err := user.NewUserModule(
		env.Ctx, // Use the test environment's context
		env.Config,
		testObservability,
		userDB,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create user module: %v", err)
	}

	// Run the router in a goroutine, managed by the routerRunCtx
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error: %v", runErr)
		}
	}()

	// Wait a moment for the router to initialize, adjust duration if needed
	time.Sleep(500 * time.Millisecond)

	// Add comprehensive cleanup function to test context to ensure proper shutdown
	t.Cleanup(func() {
		log.Println("Running user handler test cleanup...")
		// Cancel the router and event bus contexts first
		routerRunCancel()
		eventBusCancel()

		// Close the user module, which should also handle its internal dependencies
		if userModule != nil {
			if err := userModule.Close(); err != nil {
				log.Printf("Error closing User module in test cleanup: %v", err)
			}
		} else {
			// If module creation failed, ensure event bus and router are closed directly
			if eventBusImpl != nil {
				if err := eventBusImpl.Close(); err != nil {
					log.Printf("Error closing EventBus in test cleanup: %v", err)
				}
			}
			if watermillRouter != nil {
				if err := watermillRouter.Close(); err != nil {
					log.Printf("Error closing Watermill router in test cleanup: %v", err)
				}
			}
		}

		// Wait for the router goroutine to finish with a timeout
		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			log.Println("User handler router goroutine finished.")
		case <-time.After(2 * time.Second): // Use a reasonable timeout
			log.Println("WARNING: User handler router goroutine wait timed out")
		}
		log.Println("User handler test cleanup finished.")
	})

	return HandlerTestDeps{
		TestEnvironment:   env,
		UserModule:        userModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
	}
}
