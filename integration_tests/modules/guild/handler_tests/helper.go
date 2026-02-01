package guildhandlerintegrationtests

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
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	guildmodule "github.com/Black-And-White-Club/frolf-bot/app/modules/guild"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

type HandlerTestDeps struct {
	*testutils.TestEnvironment
	GuildModule *guildmodule.Module
	Router      *message.Router
	EventBus    eventbus.EventBus
	TestHelpers utils.Helpers
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing guild handler test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Guild handler test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Guild handler test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Guild handler test environment not initialized")
	}

	return testEnv
}

func SetupTestGuildHandler(t *testing.T) HandlerTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	guildDB := guilddb.NewRepository(env.DB)
	// Set the APP_ENV to "test" for the duration of the test run
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")
	watermillLogger := watermill.NopLogger{}

	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

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

	// 1. Define streams explicitly to ensure 'guild' exists
	requiredStreams := []string{"user", "discord", "leaderboard", "round", "score", "guild"}

	// 2. Create specific streams
	for _, streamName := range requiredStreams {
		if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create required NATS stream %q: %v", streamName, err)
		}
	}

	routerConfig := message.RouterConfig{CloseTimeout: 2 * time.Second}
	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	guildModule, err := guildmodule.NewGuildModule(
		env.Ctx,
		env.Config,
		observability.Observability{
			Provider: &observability.Provider{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			Registry: &observability.Registry{
				GuildMetrics: &guildmetrics.NoOpMetrics{},
				Tracer:       noop.NewTracerProvider().Tracer("test"),
			},
		},
		guildDB,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		routerRunCtx,
		env.DB,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create guild module: %v", err)
	}

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

	// 3. Robust Cleanup (LIFO Order)
	cleanup := func() {
		log.Println("Cleaning up guild handler test environment...")

		// A. Close Module FIRST
		if guildModule != nil {
			if err := guildModule.Close(); err != nil {
				log.Printf("Error closing module: %v", err)
			}
		}

		// B. Stop Router execution
		routerRunCancel()

		// C. Close Router explicitly (optional but safe)
		if watermillRouter != nil {
			_ = watermillRouter.Close()
		}

		// D. Close EventBus
		eventBusCancel()
		if eventBusImpl != nil {
			eventBusImpl.Close()
		}

		// E. Wait for goroutines
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer waitCancel()

		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			log.Println("Router goroutine finished")
		case <-waitCtx.Done():
			log.Println("WARNING: Router goroutine did not finish within timeout")
		}

		// Restore environment
		os.Setenv("APP_ENV", oldEnv)
	}
	t.Cleanup(cleanup)

	// 4. Create Environment Copy to inject local EventBus
	localEnv := *env
	localEnv.EventBus = eventBusImpl

	return HandlerTestDeps{
		TestEnvironment: &localEnv, // Use the local copy
		GuildModule:     guildModule,
		Router:          watermillRouter,
		EventBus:        eventBusImpl,
		TestHelpers:     realHelpers,
	}
}
