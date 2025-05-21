package scorehandler_integration_tests

import (
	"context"
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

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	if testEnv == nil {
		t.Fatalf("ScoreHandlers Global test environment not initialized")
	}
	return testEnv
}

func SetupTestScoreHandler(t *testing.T) ScoreHandlerTestDeps {
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

	// Truncate all relevant tables
	if err := testutils.TruncateTables(env.Ctx, env.DB, "users", "scores", "leaderboards", "rounds"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	// Use io.Discard for logs in tests for cleaner output
	discardLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	watermillLogger := watermill.NewStdLogger(false, false) // uncomment for watermill logs
	// watermillLogger := watermill.NopLogger{} // comment out for watermill logs

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

	// Create streams if they don't exist
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

	routerConfig := message.RouterConfig{
		CloseTimeout: 1 * time.Second,
	}
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
			ScoreMetrics: scoremetrics.NewNoop(),
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Logger:       discardLogger,
		},
	}

	realHelpers := utils.NewHelper(discardLogger)

	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)

	scoreDB := &scoredb.ScoreDBImpl{DB: env.DB}

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

	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error during score module tests: %v", runErr)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	t.Cleanup(func() {
		log.Println("Score Test Cleanup: Shutting down module, event bus, and router...")

		// First close the module if it was successfully created
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

		// Cancel contexts after closing resources
		eventBusCancel()
		routerRunCancel()

		// Wait for router goroutine to finish with timeout
		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()
		select {
		case <-waitCh:
			log.Println("Score router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: Score router goroutine wait timed out")
		}
		log.Println("Score Test Cleanup finished.")
	})

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
