package scoreintegrationtests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

type TestDeps struct {
	Ctx     context.Context
	DB      scoredb.ScoreDB
	BunDB   *bun.DB
	Service scoreservice.Service
	Cleanup func()
}

func SetupTestScoreService(t *testing.T) TestDeps {
	t.Helper()

	// Use the improved testutils pattern
	env := testutils.GetOrCreateTestEnv(t)

	// Use module-specific setup
	if err := env.SetupForModule("score"); err != nil {
		t.Fatalf("Failed to setup for score module: %v", err)
	}

	// Clean score tables
	if err := testutils.TruncateTables(env.Ctx, env.DB, "scores"); err != nil {
		t.Fatalf("Failed to truncate score tables: %v", err)
	}

	realDB := &scoredb.ScoreDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &scoremetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_score_service")

	// Create the ScoreService with no-op dependencies
	service := scoreservice.NewScoreService(
		realDB,
		nil, // No EventBus needed for score service
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	cleanup := func() {
		// Module cleanup is handled by SetupForModule
	}

	t.Cleanup(cleanup)

	return TestDeps{
		Ctx:     env.Ctx,
		DB:      realDB,
		BunDB:   env.DB,
		Service: service,
		Cleanup: cleanup,
	}
}
