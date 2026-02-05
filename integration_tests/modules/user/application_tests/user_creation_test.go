// integration_tests/modules/user/user_creation_test.go
package userintegrationtests

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name string
		// Modified function signatures to explicitly accept deps
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber)
		validateFn       func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
		skipCleanup      bool
	}{
		{
			name: "Success - Valid user creation",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
				}
				// No top-level error expected; check success/failure via OperationResult
				if !result.IsSuccess() {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}

				successPayload := *result.Success

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, deps.BunDB, userID, sharedtypes.GuildID("test_guild"))
				if dbErr != nil {
					t.Errorf("Failed to retrieve user %q from DB: %v", userID, dbErr)
					return
				} else if retrievedUser == nil {
					t.Fatalf("User %q not found in database after creation", userID)
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}

				if retrievedUser.User.GetUserID() != userID {
					t.Errorf("Retrieved user UserID mismatch: expected %q, got %q", userID, retrievedUser.User.GetUserID())
				}

				expectedRole := sharedtypes.UserRoleEnum("User")
				if retrievedUser.Role != expectedRole {
					t.Errorf("Retrieved user Role mismatch: expected %q, got %q", expectedRole, retrievedUser.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - Empty Discord ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("")
				tag := tagPtr(42)
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				// Service returns a domain failure payload and no top-level error for validation failures
				if err != nil {
					t.Fatalf("Did not expect top-level error for validation failure, got: %v", err)
				}
				if result.IsSuccess() {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if !result.IsFailure() {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}

				errVal := *result.Failure
				expectedErrMsg := "Discord ID cannot be empty"
				if errVal.Error() != expectedErrMsg {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedErrMsg, errVal.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Negative tag number",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(-5)
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				// Service returns a domain failure payload and no top-level error for validation failures
				if err != nil {
					t.Fatalf("Did not expect top-level error for validation failure, got: %v", err)
				}
				if result.IsSuccess() {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if !result.IsFailure() {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}

				errVal := *result.Failure
				expectedErrMsg := "tag number cannot be negative"
				if errVal.Error() != expectedErrMsg {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedErrMsg, errVal.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Success - Nil tag number",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("12345678901234568")
				var tag *sharedtypes.TagNumber = nil
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
				}
				if !result.IsSuccess() {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}

				successPayload := *result.Success

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, deps.BunDB, userID, sharedtypes.GuildID("test_guild"))
				if dbErr != nil {
					t.Errorf("Failed to retrieve user %q from DB: %v", userID, dbErr)
					return
				} else if retrievedUser == nil {
					t.Fatalf("User %q not found in database after creation", userID)
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}

				if retrievedUser.User.GetUserID() != userID {
					t.Errorf("Retrieved user UserID mismatch: expected %q, got %q", userID, retrievedUser.User.GetUserID())
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - Nil context",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)
				return nil, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				if err == nil {
					t.Fatal("Expected error for nil context, got nil")
				}
				if !result.IsFailure() {
					t.Fatalf("Result contained nil Failure payload, expected non-nil: %v", err)
				}
				if result.IsSuccess() {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}

				expectedErrMsg := "context cannot be nil"
				if err.Error() != expectedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedErrMsg, err.Error())
				}
				errVal := *result.Failure
				if errVal.Error() != expectedErrMsg {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedErrMsg, errVal.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     true,
		},
		{
			name: "Success - Large tag number",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("88888888888888888")
				tag := tagPtr(999999)
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
				}
				if !result.IsSuccess() {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}

				successPayload := *result.Success

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, deps.BunDB, userID, sharedtypes.GuildID("test_guild"))
				if dbErr != nil {
					t.Errorf("Failed to retrieve user %q from DB: %v", userID, dbErr)
					return
				} else if retrievedUser == nil {
					t.Fatalf("User with large tag number not found in database")
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - User already exists",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber) {
				userID := sharedtypes.DiscordID("99999999999999999")
				tag := tagPtr(100)
				guildID := sharedtypes.GuildID("test_guild")

				// Call CreateUser for setup and check its result
				result, err := deps.Service.CreateUser(deps.Ctx, guildID, userID, tag, nil, nil)
				if err != nil && !result.IsFailure() { // If error exists but no failure payload, something is wrong
					t.Fatalf("Failed to setup test by creating initial user: %v (Result: %+v)", err, result)
				}
				// Check if the initial creation succeeded as expected
				if !result.IsSuccess() {
					t.Fatalf("Setup failed to create initial user successfully: %+v", result.Failure)
				}
				if err != nil {
					t.Fatalf("Setup created user but returned unexpected error: %v", err)
				}

				// Return the same userID and guildID to attempt creation again in the test logic
				// This will trigger "user already exists in guild" error
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserResult, err error) {
				// Service now returns failure payload with nil top-level error for duplicate user
				if err != nil {
					t.Fatalf("Did not expect top-level error when creating duplicate user, got: %v", err)
				}
				if result.IsSuccess() {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if !result.IsFailure() {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}

				errVal := *result.Failure
				if !errors.Is(errVal, userservice.ErrUserAlreadyExists) {
					t.Errorf("Expected ErrUserAlreadyExists, got %v", errVal)
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var currentDeps TestDeps

			var ctx context.Context
			var userID sharedtypes.DiscordID
			var tag *sharedtypes.TagNumber
			var result userservice.UserResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(t)

				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				ctx, userID, tag = tc.setupFn(t, currentDeps)

			} else {
				currentDeps = TestDeps{
					Ctx: context.Background(),
					Service: userservice.NewUserService(
						nil,
						slog.New(slog.NewTextHandler(io.Discard, nil)),
						&usermetrics.NoOpMetrics{},
						noop.NewTracerProvider().Tracer("test"),
						nil,
					),
				}
				ctx, userID, tag = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.CreateUser(ctx, sharedtypes.GuildID("test_guild"), userID, tag, nil, nil)

			tc.validateFn(t, currentDeps, userID, result, err)
		})
	}
}
