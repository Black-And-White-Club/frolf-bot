package roundintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/nats-io/nats.go/jetstream"
)

type RoundTestDeps struct {
	Ctx              context.Context
	DB               rounddb.RoundDB
	BunDB            *bun.DB
	Service          roundservice.Service
	EventBus         eventbus.EventBus
	JetStreamContext jetstream.JetStream
	Cleanup          func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing round test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Round test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Round test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Round test environment not initialized")
	}

	return testEnv
}

func SetupTestRoundService(t *testing.T) RoundTestDeps {
	t.Helper()

	// Get the shared test environment
	env := GetTestEnv(t)

	// Check if containers should be recreated for stability
	if err := env.MaybeRecreateContainers(context.Background()); err != nil {
		t.Fatalf("Failed to handle container recreation: %v", err)
	}

	// Perform deep cleanup between tests for better isolation
	if err := env.DeepCleanup(); err != nil {
		t.Fatalf("Failed to perform deep cleanup: %v", err)
	}

	// Use standard stream names that the EventBus recognizes
	standardStreamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}

	// Clean up NATS consumers for relevant streams before starting the test
	if err := env.ResetJetStreamState(env.Ctx, standardStreamNames...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}

	// Clean up database tables before each test
	if err := testutils.TruncateTables(env.Ctx, env.DB, "rounds"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	realDB := &rounddb.RoundDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &roundmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_round_service")

	// Create EventBus for the service to publish events
	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx,
		env.Config.NATS.URL,
		testLogger,
		"backend", // Use standard app type that EventBus recognizes
		eventbusmetrics.NewNoop(),
		noOpTracer,
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Ensure all required streams exist after EventBus creation
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(eventBusCtx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create stream %s: %v", streamName, err)
		}
	}

	service := roundservice.NewRoundService(
		realDB,
		testLogger,
		noOpMetrics,
		noOpTracer,
		eventBusImpl,
	)

	cleanup := func() {
		if eventBusImpl != nil {
			eventBusImpl.Close()
		}
		eventBusCancel()
	}

	t.Cleanup(cleanup)

	return RoundTestDeps{
		Ctx:              env.Ctx,
		DB:               realDB,
		BunDB:            env.DB,
		Service:          service,
		EventBus:         eventBusImpl,
		JetStreamContext: env.JetStream,
		Cleanup:          cleanup,
	}
}
