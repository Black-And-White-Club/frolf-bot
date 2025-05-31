package userintegrationtests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestDeps holds dependencies needed by individual tests.
type TestDeps struct {
	Ctx     context.Context
	DB      userdb.UserDB
	BunDB   *bun.DB
	Service userservice.Service
	Cleanup func()
}

func SetupTestUserService(t *testing.T) TestDeps {
	t.Helper()

	// Use the improved testutils pattern
	env := testutils.GetOrCreateTestEnv(t)

	// Use module-specific setup
	if err := env.SetupForModule("user"); err != nil {
		t.Fatalf("Failed to setup for user module: %v", err)
	}

	// Clean user tables
	if err := testutils.TruncateTables(env.Ctx, env.DB, "users"); err != nil {
		t.Fatalf("Failed to truncate user tables: %v", err)
	}

	realDB := &userdb.UserDBImpl{DB: env.DB}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpMetrics := &usermetrics.NoOpMetrics{}
	noOpTracer := noop.NewTracerProvider().Tracer("test_user_service")

	// Create the UserService with no-op dependencies
	service := userservice.NewUserService(
		realDB,
		nil, // No EventBus needed for user service
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

// Helper function to create a tag number pointer
func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}
