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
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/nats-io/nats.go/jetstream"
)

type RoundTestDeps struct {
	Ctx              context.Context
	DB               rounddb.RoundDB
	BunDB            *bun.DB
	Service          roundservice.Service
	QueueService     roundqueue.QueueService
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
	standardStreamNames := []string{"user", "discord", "leaderboard", "round", "score"}

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

	// Create helpers for the queue service
	testHelpers := utils.NewHelper(testLogger)

	// Create queue service with the database DSN
	queueService, err := roundqueue.NewService(
		eventBusCtx,
		env.DB,
		testLogger,
		env.Config.Postgres.DSN,
		noOpMetrics,
		eventBusImpl,
		testHelpers,
	)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create queue service: %v", err)
	}

	// Start the queue service
	if err := queueService.Start(eventBusCtx); err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to start queue service: %v", err)
	}

	// Create round validator
	roundValidator := roundutil.NewRoundValidator()

	service := roundservice.NewRoundService(
		realDB,
		queueService,
		eventBusImpl,
		noOpMetrics,
		testLogger,
		noOpTracer,
		roundValidator,
	)

	cleanup := func() {
		if queueService != nil {
			queueService.Stop(context.Background())
		}
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
		QueueService:     queueService,
		EventBus:         eventBusImpl,
		JetStreamContext: env.JetStream,
		Cleanup:          cleanup,
	}
}
