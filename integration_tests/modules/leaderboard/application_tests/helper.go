package leaderboardintegrationtests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

type TestDeps struct {
	Ctx     context.Context
	DB      leaderboarddb.LeaderboardDB
	BunDB   *bun.DB
	Service leaderboardservice.Service
}

func SetupTestLeaderboardService(t *testing.T) TestDeps {
	t.Helper()

	// Use the improved testutils pattern
	env := testutils.GetOrCreateTestEnv(t)

	// Use module-specific setup (if you implement SetupForModule)
	if err := env.SetupForModule("leaderboard"); err != nil {
		t.Fatalf("Failed to setup for leaderboard module: %v", err)
	}

	realDB := &leaderboarddb.LeaderboardDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &leaderboardmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_leaderboard_service")

	service := leaderboardservice.NewLeaderboardService(
		realDB,
		nil, // No EventBus needed for leaderboard service
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	return TestDeps{
		Ctx:     env.Ctx,
		DB:      realDB,
		BunDB:   env.DB,
		Service: service,
	}
}

func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}

func boolPtr(b bool) *bool {
	return &b
}
