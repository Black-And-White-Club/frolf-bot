package roundintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"strings" // Added for testEnvOnce
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/nats-io/nats.go/jetstream" // Explicitly import jetstream here if needed for type
)

type RoundTestDeps struct {
	Ctx      context.Context
	DB       rounddb.RoundDB
	BunDB    *bun.DB
	Service  roundservice.Service
	EventBus eventbus.EventBus
	// NEW: JetStreamContext is now part of RoundTestDeps
	JetStreamContext jetstream.JetStream
	Cleanup          func()
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing round test environment...")
		var err error
		testEnv, err = testutils.NewTestEnvironment(nil) // Pass nil as *testing.T if not used in NewTestEnvironment
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Round test environment initialized successfully.")
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

	// Clean up database tables before each test
	if err := testutils.TruncateTables(env.Ctx, env.DB, "rounds"); err != nil {
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	// Clean up NATS consumers for relevant streams before starting the test
	streamNames := []string{"round", "user", "discord", "delayed"} // Added "delayed" stream for cleanup
	if err := env.ResetJetStreamState(env.Ctx, streamNames...); err != nil {
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
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
		"backend",
		eventbusmetrics.NewNoop(),
		noOpTracer,
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Create required NATS streams if they don't exist
	for _, streamName := range streamNames { // Use the updated streamNames including "delayed"
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
