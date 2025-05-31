package userhandler_integration_tests

import (
	"context"
	"io"
	"log"
	"log/slog"
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
	"go.opentelemetry.io/otel/trace/noop"
)

// HandlerTestDeps holds shared dependencies for user handler tests.
type HandlerTestDeps struct {
	*testutils.TestEnvironment
	UserModule        *user.Module
	Router            *message.Router
	EventBus          eventbus.EventBus
	ReceivedMsgs      map[string][]*message.Message
	ReceivedMsgsMutex *sync.Mutex
	TestObservability observability.Observability
	TestHelpers       utils.Helpers
}

// SetupTestUserHandler sets up the environment and dependencies for user handler tests.
func SetupTestUserHandler(t *testing.T) HandlerTestDeps {
	t.Helper()

	// Use the improved testutils pattern
	env := testutils.GetOrCreateTestEnv(t)

	// Use module-specific setup
	if err := env.SetupForModule("user"); err != nil {
		t.Fatalf("Failed to setup for user module: %v", err)
	}

	// Clean user-specific streams
	userStreams := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	if err := env.ResetJetStreamState(env.Ctx, userStreams...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}

	// Clean relevant tables
	if err := testutils.TruncateTables(env.Ctx, env.DB, "users"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	userDB := &userdb.UserDBImpl{DB: env.DB}
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

	// Ensure all required streams exist
	for _, streamName := range userStreams {
		if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
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
			UserMetrics: &usermetrics.NoOpMetrics{},
			Tracer:      noop.NewTracerProvider().Tracer("test"),
			Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Create the user module
	userModule, err := user.NewUserModule(
		env.Ctx,
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
		log.Println("Running user handler test cleanup...")
		routerRunCancel()
		eventBusCancel()

		if userModule != nil {
			if err := userModule.Close(); err != nil {
				log.Printf("Error closing User module: %v", err)
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
			log.Println("User handler router goroutine finished.")
		case <-time.After(2 * time.Second):
			log.Println("WARNING: User handler router goroutine wait timed out")
		}

		log.Println("User handler test cleanup finished.")
	}

	t.Cleanup(cleanup)

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
