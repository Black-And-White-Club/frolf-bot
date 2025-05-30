package scorehandler_integration_tests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/score"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global variables for the test environment, initialized once.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

type ScoreHandlerTestDeps struct {
	*testutils.TestEnvironment
	ScoreModule        *score.Module
	Router             *message.Router
	EventBus           eventbus.EventBus
	ReceivedMsgs       map[string][]*message.Message
	ReceivedMsgsMutex  *sync.Mutex
	PrometheusRegistry *prometheus.Registry
	TestObservability  observability.Observability
	TestHelpers        utils.Helpers
}

// GetTestEnv creates or returns the shared test environment for the test.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing score handler test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Score handler test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Score handler test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Score handler test environment not initialized")
	}

	return testEnv
}

func SetupTestScoreHandler(t *testing.T) ScoreHandlerTestDeps {
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

	// Set the APP_ENV to "test" for the duration of the test run
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Use standard stream names that the EventBus recognizes
	standardStreamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}

	// Clean up NATS consumers for all streams before starting the test
	if err := env.ResetJetStreamState(env.Ctx, standardStreamNames...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}
	log.Println("Cleaned up NATS JetStream state for score handler streams before test")

	// Truncate relevant DB tables for a clean state per test
	if err := testutils.TruncateTables(env.Ctx, env.DB, "scores", "users", "rounds", "leaderboards"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}
	log.Println("Truncated relevant tables before test")

	scoreDB := &scoredb.ScoreDBImpl{DB: env.DB}
	// Use NopLogger for quieter test logs
	watermillLogger := watermill.NopLogger{}

	// Create contexts for the event bus and router, managed by t.Cleanup
	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

	// Create the actual EventBus implementation for this test
	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend", // Use standard app type that EventBus recognizes
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Ensure all required streams exist after EventBus creation
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
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
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		Registry: &observability.Registry{
			ScoreMetrics: &scoremetrics.NoOpMetrics{},
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}

	// Use real helpers but with a discard logger
	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Create the score module
	scoreModule, err := score.NewScoreModule(
		env.Ctx,
		env.Config,
		testObservability,
		scoreDB,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create score module: %v", err)
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

	// Wait a moment for the router to initialize
	time.Sleep(500 * time.Millisecond)

	// Add comprehensive cleanup function to test context
	cleanup := func() {
		log.Println("Running score handler test cleanup...")
		// Cancel the router and event bus contexts first
		routerRunCancel()
		eventBusCancel()

		// Close the score module
		if scoreModule != nil {
			if err := scoreModule.Close(); err != nil {
				log.Printf("Error closing Score module in test cleanup: %v", err)
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
			log.Println("Score handler router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: Score handler router goroutine wait timed out")
		}

		// Restore environment
		os.Setenv("APP_ENV", oldEnv)

		log.Println("Score handler test cleanup finished.")
	}

	t.Cleanup(cleanup)

	return ScoreHandlerTestDeps{
		TestEnvironment:   env,
		ScoreModule:       scoreModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
	}
}
