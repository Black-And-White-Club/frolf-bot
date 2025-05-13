package handler_tests // Corrected package name to match directory

import (
	"context"
	"io"
	"log"
	"log/slog" // Import os for os.Exit
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
	"github.com/ThreeDotsLabs/watermill/message" // Import prometheus
	"go.opentelemetry.io/otel/trace/noop"
)

type HandlerTestDeps struct {
	*testutils.TestEnvironment
	UserModule *user.Module
	Router     *message.Router
	EventBus   eventbus.EventBus
}

func SetupTestUserHandler(t *testing.T, env *testutils.TestEnvironment) HandlerTestDeps {
	t.Helper()
	log.Printf("SetupTestUserHandler started. Received env: %v", env != nil) // Log received env value

	if env == nil {
		t.Fatalf("TestEnvironment is nil. Ensure TestMain is correctly initializing testEnv.")
	}

	// Set the APP_ENV environment variable to "test" to disable metrics in tests
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", oldEnv)
	})

	userDB := &userdb.UserDBImpl{DB: env.DB}
	watermillLogger := watermill.NewStdLogger(false, false)

	// Create the actual EventBus implementation
	eventBusImpl, err := eventbus.NewEventBus(
		env.Ctx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend", // Use a dummy appType
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Explicitly create the streams needed for user handler tests
	requiredStreams := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	for _, streamName := range requiredStreams {
		// Check if the stream exists before attempting to create it
		_, err := eventBusImpl.GetJetStream().Stream(env.Ctx, streamName)
		if err != nil && strings.Contains(err.Error(), "stream not found") {
			log.Printf("Stream %q not found, creating it.", streamName)
			if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
				eventBusImpl.Close()
				t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
			}
		} else if err != nil {
			// Handle other potential errors when checking for stream existence
			eventBusImpl.Close()
			t.Fatalf("Failed to check existence of NATS stream %q: %v", streamName, err)
		} else {
			log.Printf("Stream %q already exists, skipping creation.", streamName)
		}
	}

	// Create router with test-appropriate configuration
	routerConfig := message.RouterConfig{
		CloseTimeout: 100 * time.Millisecond,
	}
	router, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		Registry: &observability.Registry{
			UserMetrics: &usermetrics.NoOpMetrics{},
			Tracer:      noop.NewTracerProvider().Tracer("test"),
		},
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Create the user module without Prometheus metrics in tests
	userModule, err := user.NewUserModule(
		env.Ctx,
		env.Config,
		testObservability,
		userDB,
		eventBusImpl,
		router,
		realHelpers,
	)
	if err != nil {
		eventBusImpl.Close()
		router.Close()
		t.Fatalf("Failed to create user module: %v", err)
	}

	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	routerCtx, routerCancel := context.WithCancel(env.Ctx)

	go func() {
		defer routerWg.Done()
		if runErr := router.Run(routerCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error: %v", runErr)
		}
	}()

	// Wait a moment for the router to initialize
	time.Sleep(100 * time.Millisecond)

	// Add cleanup function to test context to ensure proper shutdown
	t.Cleanup(func() {
		log.Println("Running test cleanup for Watermill router...")
		// Cancel the router context first
		routerCancel()

		// Wait for the router to fully stop
		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()

		// Wait with timeout
		select {
		case <-waitCh:
			log.Println("Router shutdown completed normally")
		case <-time.After(100 * time.Millisecond):
			log.Println("WARNING: Router shutdown timed out")
		}
	})

	return HandlerTestDeps{
		TestEnvironment: env,
		UserModule:      userModule,
		Router:          router,
		EventBus:        eventBusImpl,
	}
}

func CleanupHandlerTestDeps(deps HandlerTestDeps) {
	if deps.Router != nil {
		log.Println("Stopping Watermill router...")
		if err := deps.Router.Close(); err != nil {
			log.Printf("Error closing Watermill router: %v", err)
		}
		log.Println("Watermill router stopped.")
	}
}
