package leaderboardhandler_integration_tests

import (
	"context"
	"fmt"
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
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundadapters "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/adapters"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace/noop"
)

var standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score"}

// Global variables for the test environment, initialized once.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// LeaderboardHandlerTestDeps holds shared dependencies for leaderboard handler tests.
type LeaderboardHandlerTestDeps struct {
	*testutils.TestEnvironment
	LeaderboardModule  *leaderboard.Module
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
		log.Println("Initializing leaderboard handler test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Leaderboard handler test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Leaderboard handler test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Leaderboard handler test environment not initialized")
	}

	return testEnv
}

// SetupTestLeaderboardHandler sets up the environment and dependencies for leaderboard handler tests.
func SetupTestLeaderboardHandler(t *testing.T) LeaderboardHandlerTestDeps {
	t.Helper()

	// Get the shared test environment
	env := GetTestEnv(t)

	// Reset environment for clean state
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	// Set the APP_ENV to "test" for the duration of the test run
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	realDB := leaderboarddb.NewRepository(env.DB)
	roundRepo := rounddb.NewRepository(env.DB)
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
			LeaderboardMetrics: &leaderboardmetrics.NoOpMetrics{},
			Tracer:             noop.NewTracerProvider().Tracer("test"),
			Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}

	// Use real helpers but with a discard logger
	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Initialize user service for the leaderboard module
	userRepo := userdb.NewRepository(env.DB)
	userService := userservice.NewUserService(userRepo, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, noop.NewTracerProvider().Tracer("noop"), env.DB)

	// Initialize Round Service dependencies
	userLookup := roundadapters.NewUserLookupAdapter(userRepo, env.DB)
	roundValidator := roundutil.NewRoundValidator()
	// Use NoOp metrics for round service
	roundMetrics := &roundmetrics.NoOpMetrics{}

	// Initialize Queue Service (needed for RoundService)
	// For integration tests, we might not need the actual queue running if we don't trigger jobs,
	// but we need the service instance.
	queueService, err := roundqueue.NewService(
		env.Ctx,
		env.DB,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		env.Config.Postgres.DSN,
		roundMetrics,
		eventBusImpl,
		realHelpers,
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create round queue service: %v", err)
	}

	// Initialize Round Service
	roundService := roundservice.NewRoundService(
		roundRepo,
		queueService,
		eventBusImpl,
		userLookup,
		roundMetrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		noop.NewTracerProvider().Tracer("noop"),
		roundValidator,
		env.DB,
	)

	// Create the leaderboard module
	leaderboardModule, err := leaderboard.NewLeaderboardModule(
		env.Ctx,
		env.Config,
		testObservability,
		env.DB,
		realDB,
		roundService,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
		eventBusImpl.GetJetStream(),
		userService,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create leaderboard module: %v", err)
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

	// Wait for router to be running
	select {
	case <-watermillRouter.Running():
		// ready
	case <-time.After(5 * time.Second):
		t.Fatal("router failed to start")
	}

	// Add comprehensive cleanup function to test context
	cleanup := func() {
		log.Println("Running leaderboard handler test cleanup...")

		// Close the leaderboard module first to allow graceful router shutdown
		// while dependencies (EventBus, NATS) are still active.
		if leaderboardModule != nil {
			if err := leaderboardModule.Close(); err != nil {
				log.Printf("Error closing Leaderboard module in test cleanup: %v", err)
			}
		} else {
			// If module creation failed, ensure event bus and router are closed directly
			if watermillRouter != nil {
				if err := watermillRouter.Close(); err != nil {
					log.Printf("Error closing Watermill router in test cleanup: %v", err)
				}
			}
		}

		// Cancel the router and event bus contexts after module closure
		routerRunCancel()
		eventBusCancel()

		// Ensure event bus is closed directly if not already closed by router
		if eventBusImpl != nil {
			if err := eventBusImpl.Close(); err != nil {
				// Ignore errors if it's already closed
				log.Printf("Note: Error closing EventBus in test cleanup (might be already closed): %v", err)
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
			log.Println("Leaderboard handler router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: Leaderboard handler router goroutine wait timed out")
		}

		// Restore environment
		os.Setenv("APP_ENV", oldEnv)

		log.Println("Leaderboard handler test cleanup finished.")
	}

	t.Cleanup(cleanup)

	// Create a shallow copy of the environment to avoid modifying the global one
	// and inject the local EventBus so RunTest uses the correct connection
	localEnv := *env
	localEnv.EventBus = eventBusImpl

	return LeaderboardHandlerTestDeps{
		TestEnvironment:   &localEnv,
		LeaderboardModule: leaderboardModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
	}
}

// Helper functions
func tagPtr(n sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &n
}

func boolPtr(b bool) *bool {
	return &b
}

// WaitForMessageProcessed waits for a signal on a channel indicating a message has been processed.
func WaitForMessageProcessed(msgProcessedChan <-chan struct{}, timeout time.Duration) error {
	select {
	case <-msgProcessedChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for message to be processed")
	}
}
