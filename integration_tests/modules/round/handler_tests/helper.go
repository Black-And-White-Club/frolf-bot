package roundhandler_integration_tests

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
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace/noop"
)

// Ensure these cover all streams the Round module interacts with
var standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score"}

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

type RoundHandlerTestDeps struct {
	*testutils.TestEnvironment
	RoundModule       *round.Module
	Router            *message.Router
	EventBus          eventbus.EventBus
	ReceivedMsgs      map[string][]*message.Message
	ReceivedMsgsMutex *sync.Mutex
	TestObservability observability.Observability
	TestHelpers       utils.Helpers
}

// HandlerTestDeps alias for consistency
type HandlerTestDeps = RoundHandlerTestDeps

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	testEnvOnce.Do(func() {
		log.Println("Initializing round handler test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Round handler test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Round handler test environment initialization failed: %v", testEnvErr)
	}
	if testEnv == nil {
		t.Fatalf("Round handler test environment not initialized")
	}

	return testEnv
}

func SetupTestRoundHandler(t *testing.T) RoundHandlerTestDeps {
	t.Helper()

	// 1. Use NopLogger to silence Watermill noise
	watermillLogger := watermill.NopLogger{}

	env := GetTestEnv(t)

	// 2. Prepare Contexts
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	testCtx, testCancel := context.WithCancel(context.Background())
	routerRunCtx, routerRunCancel := context.WithCancel(testCtx)

	// 3. Create Isolated EventBus (Already using io.Discard, which is good)
	eventBusImpl, err := eventbus.NewEventBus(
		testCtx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend",
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		testCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// 4. Create Streams (Add error checking here but keep it quiet)
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(testCtx, streamName); err != nil {
			t.Fatalf("Failed to create stream %s: %v", streamName, err)
		}
	}

	// 5. Router Setup
	watermillRouter, err := message.NewRouter(message.RouterConfig{
		CloseTimeout: 1 * time.Second,
	}, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		testCancel()
		t.Fatalf("Failed to create router: %v", err)
	}
	watermillRouter.AddMiddleware(middleware.CorrelationID)

	// 6. Dependencies & Module
	realDB := rounddb.NewRepository(env.DB)
	userRepo := userdb.NewRepository(env.DB)
	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	roundModule, err := round.NewRoundModule(
		testCtx,
		env.Config,
		observability.Observability{
			Provider: &observability.Provider{Logger: slog.New(slog.NewTextHandler(os.Stdout, nil))},
			Registry: &observability.Registry{RoundMetrics: &roundmetrics.NoOpMetrics{}, Tracer: noop.NewTracerProvider().Tracer("test")},
		},
		realDB,
		env.DB,
		userRepo,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
	)
	if err != nil {
		eventBusImpl.Close()
		testCancel()
		t.Fatalf("Failed to create round module: %v", err)
	}

	// 7. Run Router
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			// Use t.Errorf so it only shows up if something actually goes wrong
			t.Errorf("Router exited unexpectedly: %v", runErr)
		}
	}()

	select {
	case <-watermillRouter.Running():
	case <-time.After(5 * time.Second):
		t.Fatal("Router startup timed out")
	}

	// 8. Silent Cleanup
	cleanup := func() {
		// We removed the log.Println statements here.
		// If you need to debug a hang again, you can add t.Log("Cleaning up...")
		if watermillRouter != nil {
			_ = watermillRouter.Close()
		}

		if roundModule != nil {
			_ = roundModule.Close()
		}

		if eventBusImpl != nil {
			eventBusImpl.Close()
		}

		routerRunCancel()
		testCancel()

		routerWg.Wait()
		os.Setenv("APP_ENV", oldEnv)
	}
	t.Cleanup(cleanup)

	localEnv := *env
	localEnv.EventBus = eventBusImpl

	return RoundHandlerTestDeps{
		TestEnvironment:   &localEnv,
		RoundModule:       roundModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestHelpers:       realHelpers,
	}
}
func boolPtr(b bool) *bool {
	return &b
}
