package roundintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundadapters "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/adapters"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	testEnv             *testutils.TestEnvironment
	testEnvErr          error
	testEnvOnce         sync.Once
	standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score"}
)

type RoundTestDeps struct {
	Ctx              context.Context
	DB               rounddb.Repository
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

	// Reset environment for clean state with timeout
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	realDB := rounddb.NewRepository(env.DB)

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &roundmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_round_service")

	// Create context for this test setup
	testCtx, testCancel := context.WithCancel(env.Ctx)

	// Use shared EventBus
	eventBusImpl := env.EventBus

	// Ensure all required streams exist
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(testCtx, streamName); err != nil {
			// Ignore if stream already exists
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "stream name already in use") {
				testCancel()
				t.Fatalf("Failed to create stream %s: %v", streamName, err)
			}
		}
	}

	// Create helpers for the queue service
	testHelpers := utils.NewHelper(testLogger)

	// Create queue service with test-optimized settings (fast polling for quick job execution)
	fastPollInterval := 100 * time.Millisecond
	queueService, err := roundqueue.NewServiceWithOptions(
		testCtx,
		env.DB,
		testLogger,
		env.Config.Postgres.DSN,
		noOpMetrics,
		eventBusImpl,
		testHelpers,
		&roundqueue.ServiceOptions{
			FetchPollInterval: &fastPollInterval,
		},
	)
	if err != nil {
		testCancel()
		t.Fatalf("Failed to create queue service: %v", err)
	}

	// Start the queue service
	if err := queueService.Start(testCtx); err != nil {
		testCancel()
		t.Fatalf("Failed to start queue service: %v", err)
	}

	// Give River a moment to fully initialize its polling loop
	time.Sleep(200 * time.Millisecond)

	// Create round validator
	roundValidator := roundutil.NewRoundValidator()

	// Create user repository for the adapter
	userRepository := userdb.NewRepository(env.DB)

	service := roundservice.NewRoundService(
		realDB,
		queueService,
		eventBusImpl,
		roundadapters.NewUserLookupAdapter(userRepository, env.DB),
		noOpMetrics,
		testLogger,
		noOpTracer,
		roundValidator,
		env.DB,
	)

	cleanup := func() {
		if queueService != nil {
			queueService.Stop(context.Background())
		}
		// Do NOT close shared EventBus
		testCancel()
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
