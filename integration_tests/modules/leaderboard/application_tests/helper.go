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

var testEnv *testutils.TestEnvironment

type TestDeps struct {
	Ctx     context.Context
	DB      leaderboarddb.LeaderboardDB
	BunDB   *bun.DB
	Service leaderboardservice.Service
	Cleanup func()
}

func SetupTestLeaderboardService(ctx context.Context, db *bun.DB, t *testing.T) TestDeps {
	t.Helper()

	realDB := &leaderboarddb.LeaderboardDBImpl{DB: db}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &leaderboardmetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_leaderboard_service")

	service := leaderboardservice.NewLeaderboardService(
		realDB,
		nil,
		testLogger,
		noOpMetrics,
		noOpTracer,
	)

	return TestDeps{
		Ctx:     ctx,
		DB:      realDB,
		BunDB:   db,
		Service: service,
		Cleanup: func() {
		},
	}
}

func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}

func boolPtr(b bool) *bool {
	return &b
}
