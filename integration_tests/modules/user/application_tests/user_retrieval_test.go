// integration_tests/modules/user/user_retrieval_test.go
package userintegrationtests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetUser(t *testing.T) {
	tests := []struct {
		name             string
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID)
		validateFn       func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
		skipCleanup      bool
	}{
		{
			name: "Success - User exists",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)

				// Create a user first
				result, err := deps.Service.CreateUser(deps.Ctx, userID, tag)
				if err != nil {
					t.Fatalf("Failed to create user for test setup: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("User creation failed in setup: %+v", result.Failure)
				}

				return deps.Ctx, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("GetUser returned unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Result contained unexpected Error: %v", result.Error)
				}
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*userevents.GetUserResponsePayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.GetUserResponsePayload")
				}

				if successPayload.User == nil {
					t.Fatalf("Success payload contained nil User data")
				}
				if successPayload.User.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.User.UserID)
				}

				expectedRole := sharedtypes.UserRoleEnum("Rattler")
				if successPayload.User.Role != expectedRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", expectedRole, successPayload.User.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - User does not exist",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("99999999999999999")

				// Ensure user doesn't exist
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}

				return deps.Ctx, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for non-existent user, got nil")
				}
				if result.Error == nil {
					t.Fatalf("Result contained nil Error, expected non-nil: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}

				failurePayload, ok := result.Failure.(*userevents.GetUserFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.GetUserFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}

				expectedWrappedErrMsg := "GetUser operation failed: user not found"
				if err.Error() != expectedWrappedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedWrappedErrMsg, err.Error())
				}

				expectedOriginalErrMsg := "user not found"
				if result.Error.Error() != expectedOriginalErrMsg {
					t.Errorf("Result error message mismatch: expected %q, got %q", expectedOriginalErrMsg, result.Error.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Empty Discord ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				return deps.Ctx, sharedtypes.DiscordID("")
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for empty Discord ID, got nil")
				}
				if result.Error == nil {
					t.Fatalf("Result contained nil Error, expected non-nil: %v", err)
				}

				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Nil context",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("12345678901234567")
				return nil, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for nil context, got nil")
				}
			},
			expectedSuccess: false,
			skipCleanup:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var currentDeps TestDeps
			var ctx context.Context
			var userID sharedtypes.DiscordID
			var result userservice.UserOperationResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(sharedCtx, sharedDB, t)

				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				ctx, userID = tc.setupFn(t, currentDeps)
			} else {
				currentDeps = TestDeps{
					Ctx: context.Background(),
					Service: userservice.NewUserService(
						nil,
						nil,
						slog.New(slog.NewTextHandler(io.Discard, nil)),
						&usermetrics.NoOpMetrics{},
						noop.NewTracerProvider().Tracer("test"),
					),
				}
				ctx, userID = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.GetUser(ctx, userID)

			tc.validateFn(t, currentDeps, userID, result, err)
		})
	}
}

func TestGetUserRole(t *testing.T) {
	tests := []struct {
		name             string
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID)
		validateFn       func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
		skipCleanup      bool
	}{
		{
			name: "Success - Valid user role",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)

				// Create a user first
				result, err := deps.Service.CreateUser(deps.Ctx, userID, tag)
				if err != nil {
					t.Fatalf("Failed to create user for test setup: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("User creation failed in setup: %+v", result.Failure)
				}

				return deps.Ctx, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("GetUserRole returned unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Result contained unexpected Error: %v", result.Error)
				}
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.GetUserRoleResponsePayload")
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}

				expectedRole := sharedtypes.UserRoleEnum("Rattler")
				if successPayload.Role != expectedRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", expectedRole, successPayload.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - User does not exist",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("99999999999999999")

				// Ensure user doesn't exist
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}

				return deps.Ctx, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for non-existent user, got nil")
				}
				if result.Error == nil {
					t.Fatalf("Result contained nil Error, expected non-nil: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}

				failurePayload, ok := result.Failure.(*userevents.GetUserRoleFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.GetUserRoleFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}

				expectedErrMsgSubstring := "GetUserRole operation failed: failed to retrieve user role"
				if err.Error() != expectedErrMsgSubstring {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedErrMsgSubstring, err.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},

		{
			name: "Failure - Nil context",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID) {
				userID := sharedtypes.DiscordID("12345678901234567")
				return nil, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for nil context, got nil")
				}
				// The specific error validation would depend on how your service wrapper handles nil contexts
			},
			expectedSuccess: false,
			skipCleanup:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var currentDeps TestDeps
			var ctx context.Context
			var userID sharedtypes.DiscordID
			var result userservice.UserOperationResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(sharedCtx, sharedDB, t)

				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				ctx, userID = tc.setupFn(t, currentDeps)
			} else {
				currentDeps = TestDeps{
					Ctx: context.Background(),
					Service: userservice.NewUserService(
						nil,
						nil,
						slog.New(slog.NewTextHandler(io.Discard, nil)),
						&usermetrics.NoOpMetrics{},
						noop.NewTracerProvider().Tracer("test"),
					),
				}
				ctx, userID = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.GetUserRole(ctx, userID)

			tc.validateFn(t, currentDeps, userID, result, err)
		})
	}
}
