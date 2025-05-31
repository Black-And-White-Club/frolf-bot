package leaderboardhandler_integration_tests

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
)

// LeaderboardHandlerTestDeps holds shared dependencies for leaderboard handler tests.
type LeaderboardHandlerTestDeps struct {
	*testutils.TestEnvironment
	LeaderboardModule *leaderboard.Module
	Router            *message.Router
	EventBus          eventbus.EventBus
	ReceivedMsgs      map[string][]*message.Message
	ReceivedMsgsMutex *sync.Mutex
	TestObservability observability.Observability
	TestHelpers       utils.Helpers
}

// SetupTestLeaderboardHandler sets up the environment and dependencies for leaderboard handler tests.
func SetupTestLeaderboardHandler(t *testing.T) LeaderboardHandlerTestDeps {
	t.Helper()

	// Use the improved testutils pattern
	env := testutils.GetOrCreateTestEnv(t)

	// Use module-specific setup
	if err := env.SetupForModule("leaderboard"); err != nil {
		t.Fatalf("Failed to setup for leaderboard module: %v", err)
	}

	// Clean leaderboard-specific streams
	leaderboardStreams := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	if err := env.ResetJetStreamState(env.Ctx, leaderboardStreams...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}

	leaderboardDB := &leaderboarddb.LeaderboardDBImpl{DB: env.DB}
	watermillLogger := watermill.NopLogger{}

	// Create contexts for the event bus and router
	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

	// Create the EventBus
	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend",
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Ensure streams exist
	for _, streamName := range leaderboardStreams {
		if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create NATS stream %q: %v", streamName, err)
		}
	}

	// Create router
	routerConfig := message.RouterConfig{CloseTimeout: 1 * time.Second}
	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	// Test observability
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

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Create the leaderboard module
	leaderboardModule, err := leaderboard.NewLeaderboardModule(
		env.Ctx,
		env.Config,
		testObservability,
		leaderboardDB,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create leaderboard module: %v", err)
	}

	// Run the router in a goroutine
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error: %v", runErr)
		}
	}()

	// Wait for router to initialize
	time.Sleep(500 * time.Millisecond)

	// Cleanup function
	cleanup := func() {
		log.Println("Running leaderboard handler test cleanup...")
		routerRunCancel()
		eventBusCancel()

		if leaderboardModule != nil {
			if err := leaderboardModule.Close(); err != nil {
				log.Printf("Error closing Leaderboard module: %v", err)
			}
		}

		// Wait for router goroutine with timeout
		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			log.Println("Router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: Router goroutine wait timed out")
		}

		log.Println("Leaderboard handler test cleanup finished.")
	}

	t.Cleanup(cleanup)

	return LeaderboardHandlerTestDeps{
		TestEnvironment:   env,
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
