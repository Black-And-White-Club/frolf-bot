package scorehandler_integration_tests

import (
	"context"
	"io"
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
	"go.opentelemetry.io/otel/trace/noop"
)

// ScoreHandlerTestDeps holds dependencies needed for score handler tests.
type ScoreHandlerTestDeps struct {
	*testutils.TestEnvironment
	ScoreModule *score.Module
	Router      *message.Router
	EventBus    eventbus.EventBus
}

// SetupTestScoreHandler sets up the environment and dependencies for score handler tests.
func SetupTestScoreHandler(t *testing.T, env *testutils.TestEnvironment) ScoreHandlerTestDeps {
	t.Helper()

	if env == nil {
		t.Fatalf("TestEnvironment is nil. Ensure TestMain is correctly initializing testEnv.")
	}

	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")
	t.Cleanup(func() {
		os.Setenv("APP_ENV", oldEnv)
	})

	scoreDB := &scoredb.ScoreDBImpl{DB: env.DB}
	watermillLogger := watermill.NewStdLogger(false, false)

	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
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

	requiredStreams := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	for _, streamName := range requiredStreams {
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
		CloseTimeout: 5 * time.Second,
	}
	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		Registry: &observability.Registry{
			ScoreMetrics: &scoremetrics.NoOpMetrics{},
			Tracer:       noop.NewTracerProvider().Tracer("test"),
		},
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

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
		if eventBusImpl != nil {
			if err := eventBusImpl.Close(); err != nil {
				log.Printf("Error closing EventBus in test cleanup: %v", err)
			}
		}
		if scoreModule != nil {
			if err := scoreModule.Close(); err != nil {
				log.Printf("Error closing Score module in test cleanup: %v", err)
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
		case <-time.After(1 * time.Second):
			log.Println("WARNING: Score module Router shutdown timed out")
		}
	})

	return ScoreHandlerTestDeps{
		TestEnvironment: env,
		ScoreModule:     scoreModule,
		Router:          watermillRouter,
		EventBus:        eventBusImpl,
	}
}
