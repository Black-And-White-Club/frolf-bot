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
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
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

// Shared dependencies
var (
	sharedDeps     *RoundHandlerTestDeps
	sharedDepsOnce sync.Once
	sharedCleanup  func()
)

// CleanupSharedDeps cleans up the shared dependencies.
// Should be called from TestMain.
func CleanupSharedDeps() {
	if sharedCleanup != nil {
		sharedCleanup()
		sharedCleanup = nil
	}
}

func SetupTestRoundHandler(t *testing.T) RoundHandlerTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	// Soft reset between tests
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer resetCancel()
	if err := env.SoftReset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	sharedDepsOnce.Do(func() {
		log.Println("Initializing shared round handler dependencies...")
		// 1. Use NopLogger to silence Watermill noise
		watermillLogger := watermill.NopLogger{}

		oldEnv := os.Getenv("APP_ENV")
		os.Setenv("APP_ENV", "test")

		// Helper context that lives as long as the shared deps
		// We use a background context because this setup outlives the individual test 't'
		globalCtx, globalCancel := context.WithCancel(context.Background())
		routerRunCtx, routerRunCancel := context.WithCancel(globalCtx)

		// 3. Create Isolated EventBus
		eventBusImpl, err := eventbus.NewEventBus(
			globalCtx,
			env.Config.NATS.URL,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			"backend",
			&eventbusmetrics.NoOpMetrics{},
			noop.NewTracerProvider().Tracer("test"),
		)
		if err != nil {
			globalCancel()
			log.Fatalf("Failed to create EventBus: %v", err)
		}

		// 4. Create Streams
		for _, streamName := range standardStreamNames {
			// CreateStream is idempotentish or we just ignore if it exists?
			// The simpler way is ensuring it exists.
			if err := eventBusImpl.CreateStream(globalCtx, streamName); err != nil {
				// We might get an error if stream exists, which is fine in a shared env logic
				// but since we start fresh container in GetTestEnv, it should be fine.
				// However, if CreateStream fails for other reasons, it's bad.
				// Let's log and proceed, or fail?
				// Since testEnv is fresh, streams shouldn't exist unless created by something else.
				log.Printf("CreateStream %s result: %v", streamName, err)
			}
		}

		// 5. Router Setup
		watermillRouter, err := message.NewRouter(message.RouterConfig{
			CloseTimeout: 1 * time.Second,
		}, watermillLogger)
		if err != nil {
			eventBusImpl.Close()
			globalCancel()
			log.Fatalf("Failed to create router: %v", err)
		}
		watermillRouter.AddMiddleware(middleware.CorrelationID)

		// 6. Dependencies & Module
		realDB := rounddb.NewRepository(env.DB)
		userRepo := userdb.NewRepository(env.DB)
		realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		userService := userservice.NewUserService(userRepo, logger, nil, noop.NewTracerProvider().Tracer("noop"), env.DB)

		roundModule, err := round.NewRoundModule(
			globalCtx,
			env.Config,
			observability.Observability{
				Provider: &observability.Provider{Logger: slog.New(slog.NewTextHandler(os.Stdout, nil))},
				Registry: &observability.Registry{RoundMetrics: &roundmetrics.NoOpMetrics{}, Tracer: noop.NewTracerProvider().Tracer("test")},
			},
			realDB,
			env.DB,
			userRepo,
			userService,
			eventBusImpl,
			watermillRouter,
			realHelpers,
			routerRunCtx,
		)
		if err != nil {
			eventBusImpl.Close()
			globalCancel()
			log.Fatalf("Failed to create round module: %v", err)
		}

		// 7. Run Router
		routerWg := &sync.WaitGroup{}
		routerWg.Add(1)
		go func() {
			defer routerWg.Done()
			if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
				log.Printf("Router exited unexpectedly: %v", runErr)
			}
		}()

		// Wait for router to start
		select {
		case <-watermillRouter.Running():
		case <-time.After(5 * time.Second):
			log.Fatal("Router startup timed out")
		}

		// Save the cleanup function
		sharedCleanup = func() {
			log.Println("Cleaning up shared round handler dependencies...")
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
			globalCancel()
			routerWg.Wait()
			os.Setenv("APP_ENV", oldEnv)
		}

		localEnv := *env
		localEnv.EventBus = eventBusImpl

		sharedDeps = &RoundHandlerTestDeps{
			TestEnvironment:   &localEnv,
			RoundModule:       roundModule,
			Router:            watermillRouter,
			EventBus:          eventBusImpl,
			ReceivedMsgs:      make(map[string][]*message.Message),
			ReceivedMsgsMutex: &sync.Mutex{},
			TestHelpers:       realHelpers,
		}
	})

	if sharedDeps == nil {
		t.Fatal("Failed to initialize shared dependencies (sharedDeps is nil)")
	}

	// Just reuse the shared dependencies.
	// Since TestEnvironment is a pointer in the struct, and we only lightly modified it (EventBus),
	// it should be fine.
	return *sharedDeps
}
func boolPtr(b bool) *bool {
	return &b
}
