package leaderboardhandler_integration_tests

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
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

// GetTestEnv creates a new test environment for the test.
// This ensures each test gets a clean environment.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	if testEnv == nil {
		t.Fatalf("LeaderboardHandlers Global test environment not initialized")
	}
	return testEnv
}

// SetupTestLeaderboardHandler sets up the environment and dependencies for leaderboard handler tests.
// It creates a new router and event bus instance for each test function's scope.
func SetupTestLeaderboardHandler(t *testing.T) LeaderboardHandlerTestDeps {
	t.Helper()

	// Get a fresh test environment for this test
	env := GetTestEnv(t)

	// Set the APP_ENV to "test" for the duration of the test run
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", oldEnv)
	})

	// Clean up NATS consumers for all streams before starting the test
	streamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	if err := env.ResetJetStreamState(env.Ctx, streamNames...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}
	log.Println("Cleaned up NATS JetStream state for all streams before test")

	if err := testutils.TruncateTables(env.Ctx, env.DB, "users", "scores", "leaderboards", "rounds"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// watermillLogger := watermill.NewStdLogger(false, false) // uncomment for watermill logs
	watermillLogger := watermill.NopLogger{} // comment out for watermill logs

	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)

	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx,
		env.Config.NATS.URL,
		discardLogger,
		"backend",
		eventbusmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	for _, streamName := range streamNames {
		_, err := eventBusImpl.GetJetStream().Stream(env.Ctx, streamName)
		if err != nil && strings.Contains(err.Error(), "stream not found") {
			if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
				eventBusImpl.Close()
				eventBusCancel()
				t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
			}
		} else if err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to check existence of NATS stream %q: %v", streamName, err)
		}
	}

	routerConfig := message.RouterConfig{CloseTimeout: 1 * time.Second}

	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: discardLogger,
		},
		Registry: &observability.Registry{
			LeaderboardMetrics: leaderboardmetrics.NewNoop(),
			Tracer:             noop.NewTracerProvider().Tracer("test"),
			Logger:             discardLogger,
		},
	}
	realHelpers := utils.NewHelper(discardLogger)

	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)

	leaderboardDB := &leaderboarddb.LeaderboardDBImpl{DB: env.DB}

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

	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error: %v", runErr)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	t.Cleanup(func() {
		log.Println("Leaderboard Test Cleanup: Shutting down module, event bus, and router...")
		if leaderboardModule != nil {
			if err := leaderboardModule.Close(); err != nil {
				log.Printf("Error closing Leaderboard module in test cleanup: %v", err)
			}
		} else {
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

		eventBusCancel()
		routerRunCancel()

		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()
		select {
		case <-waitCh:
			log.Println("Leaderboard router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: Leaderboard router goroutine wait timed out")
		}
		log.Println("Leaderboard Test Cleanup finished.")
	})

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

// Helper functions and dummy types (replace with your actual implementations)
func tagPtr(n sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &n
}

func boolPtr(b bool) *bool {
	return &b
}

// sanitizeForNATS sanitizes a string for use in NATS topics or durable names.
func sanitizeForNATS(s string) string {
	sanitized := strings.ReplaceAll(s, ".", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	sanitized = reg.ReplaceAllString(sanitized, "")
	sanitized = strings.Trim(sanitized, "-_")
	return sanitized
}

func sortLeaderboardData(data leaderboardtypes.LeaderboardData) {
	slices.SortFunc(data, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0
		}
		if a.TagNumber == 0 {
			return -1
		}
		if b.TagNumber == 0 {
			return 1
		}
		return int(a.TagNumber - b.TagNumber)
	})
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
