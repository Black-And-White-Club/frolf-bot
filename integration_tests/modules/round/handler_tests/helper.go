package roundhandler_integration_tests

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

// var (
// 	testEnv     *testutils.TestEnvironment
// 	testEnvOnce sync.Once
// 	testEnvErr  error
// )

var standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score"}

type RoundHandlerTestDeps struct {
	*testutils.TestEnvironment
	RoundModule       *round.Module
	Router            *message.Router
	EventBus          eventbus.EventBus
	ReceivedMsgs      map[string][]*message.Message
	ReceivedMsgsMutex *sync.Mutex
	TestObservability observability.Observability
	TestHelpers       utils.Helpers
	MessageCapture    *testutils.MessageCapture
}

// HandlerTestDeps is an alias for RoundHandlerTestDeps for consistency with user module naming
type HandlerTestDeps = RoundHandlerTestDeps

// func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
// 	t.Helper()
// 	testEnvOnce.Do(func() {
// 		env, err := testutils.NewTestEnvironment(t)
// 		if err != nil {
// 			testEnvErr = err
// 			return
// 		}
// 		testEnv = env
// 	})
// 	if testEnvErr != nil {
// 		t.Fatalf("failed to initialize test environment: %v", testEnvErr)
// 	}
// 	return testEnv
// }

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error

	// Global shared deps for the package
	sharedDeps RoundHandlerTestDeps
	// sharedOnce sync.Once
)

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	testEnvOnce.Do(func() {
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			return
		}
		testEnv = env
	})
	if testEnvErr != nil {
		t.Fatalf("failed to initialize test environment: %v", testEnvErr)
	}
	return testEnv
}

// initSharedInfrastructure initializes the components that should persist
// across all tests in this package.
func initSharedInfrastructure(ctx context.Context, env *testutils.TestEnvironment) (RoundHandlerTestDeps, error) {
	os.Setenv("APP_ENV", "test")

	// 1. Create EventBus
	eventBusImpl, err := eventbus.NewEventBus(
		ctx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend",
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		return RoundHandlerTestDeps{}, fmt.Errorf("failed to create EventBus: %w", err)
	}

	// 2. Setup Streams
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(ctx, streamName); err != nil {
			eventBusImpl.Close()
			return RoundHandlerTestDeps{}, fmt.Errorf("failed to create stream %q: %w", streamName, err)
		}
	}

	// 3. Setup Router
	watermillRouter, err := message.NewRouter(message.RouterConfig{
		CloseTimeout: 5 * time.Second,
	}, watermill.NopLogger{})
	if err != nil {
		eventBusImpl.Close()
		return RoundHandlerTestDeps{}, fmt.Errorf("failed to create router: %w", err)
	}
	watermillRouter.AddMiddleware(middleware.CorrelationID)

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
		},
		Registry: &observability.Registry{
			RoundMetrics: &roundmetrics.NoOpMetrics{},
			Tracer:       noop.NewTracerProvider().Tracer("test"),
		},
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// 4. Initialize Round Module
	// Note: We pass ctx twice (for the module and the router runner context)
	roundModule, err := round.NewRoundModule(
		ctx,
		env.Config,
		testObservability,
		&rounddb.RoundDBImpl{DB: env.DB},
		&userdb.UserDBImpl{DB: env.DB},
		eventBusImpl,
		watermillRouter,
		realHelpers,
		ctx,
	)
	if err != nil {
		eventBusImpl.Close()
		watermillRouter.Close()
		return RoundHandlerTestDeps{}, fmt.Errorf("failed to create round module: %w", err)
	}

	// 5. Start the Router goroutine
	// It will stay running until the TestMain context is cancelled
	go func() {
		if runErr := watermillRouter.Run(ctx); runErr != nil && runErr != context.Canceled {
			log.Printf("Global Watermill router exited with error: %v", runErr)
		}
	}()

	// Give a small window for the router to finish subscribing
	time.Sleep(500 * time.Millisecond)

	return RoundHandlerTestDeps{
		TestEnvironment:   env,
		RoundModule:       roundModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
	}, nil
}

// SetupTestRoundHandler is called by individual tests.
// It resets the database but reuses the shared infrastructure.
func SetupTestRoundHandler(t *testing.T) RoundHandlerTestDeps {
	t.Helper()

	if testEnv == nil {
		t.Fatal("testEnv not initialized. Ensure TestMain is calling initSharedInfrastructure.")
	}

	// 1. Reset Database State for isolation
	resetCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := testEnv.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset database for test: %v", err)
	}

	// 2. Clear the mutex-protected map if you are using it for manual tracking
	// (though RunTest handles its own tracking usually)
	sharedDeps.ReceivedMsgsMutex.Lock()
	sharedDeps.ReceivedMsgs = make(map[string][]*message.Message)
	sharedDeps.ReceivedMsgsMutex.Unlock()

	return sharedDeps
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
