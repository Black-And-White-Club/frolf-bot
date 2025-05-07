// integration_tests/modules/user/helper.go
package userintegrationtests

import (
	"context"
	"io" // Import io for io.Discard
	"log/slog"
	"testing"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/uptrace/bun"

	"go.opentelemetry.io/otel/trace/noop"
)

// testEnv is the shared test environment managed by TestMain.
var testEnv *testutils.TestEnvironment

// TestDeps holds dependencies needed by individual tests.
type TestDeps struct {
	Ctx     context.Context
	DB      userdb.UserDB
	BunDB   *bun.DB
	Service userservice.Service
	Cleanup func()
}

// SetupTestUserService provides dependencies for individual tests using the shared environment.
// It no longer creates or cleans up the test environment itself.
func SetupTestUserService(ctx context.Context, db *bun.DB, t *testing.T) TestDeps {
	t.Helper()

	// Use the provided context and DB
	realDB := &userdb.UserDBImpl{DB: db}

	// Create the UserService using the provided DB and mock/no-op dependencies
	service := userservice.NewUserService(
		realDB,
		nil, // Assuming nil for EventBus
		slog.New(slog.NewTextHandler(io.Discard, nil)), // Silent logger
		&usermetrics.NoOpMetrics{},                     // No-op metrics
		noop.NewTracerProvider().Tracer("test"),        // No-op tracer
	)

	// Return the dependencies. Cleanup is a no-op for tests using this setup.
	return TestDeps{
		Ctx:     ctx, // Use the provided context
		DB:      realDB,
		BunDB:   db, // Provide the provided bun.DB
		Service: service,
		Cleanup: func() {}, // No-op cleanup
	}
}

// Helper function to create a tag number pointer
// Assuming this function is defined in this file or another file in this package.
func tagPtr(val int) *sharedtypes.TagNumber {
	tag := sharedtypes.TagNumber(val)
	return &tag
}
