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

var testEnv *testutils.TestEnvironment

type TestDeps struct {
	Ctx     context.Context
	DB      scoredb.ScoreDB
	BunDB   *bun.DB
	Service scoreservice.Service
	Cleanup func()
}

func SetupTestScoreService(ctx context.Context, db *bun.DB, t *testing.T) TestDeps {
	t.Helper()

	realDB := &scoredb.ScoreDBImpl{DB: db}

	service := scoreservice.NewScoreService(
		realDB,
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&scoremetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test_score_service"),
	)

	return TestDeps{
		Ctx:     ctx,
		DB:      realDB,
		BunDB:   db,
		Service: service,
		Cleanup: func() {},
	}
}
