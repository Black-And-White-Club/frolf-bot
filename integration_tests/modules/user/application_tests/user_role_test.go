package userintegrationtests

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestUpdateUserRoleInDatabase(t *testing.T) {
	tests := []struct {
		name            string
		setupFn         func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, sharedtypes.UserRoleEnum)
		validateFn      func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum, result userservice.UserOperationResult, err error)
		expectedSuccess bool
		skipCleanup     bool
	}{
		{
			name: "Success - Update user role",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, sharedtypes.UserRoleEnum) {
				userID := sharedtypes.DiscordID("123456789012345678")
				newRole := sharedtypes.UserRoleAdmin

				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}

				// Create a user first
				// Assuming CreateUser exists and works in your service
				createResult, createErr := deps.Service.CreateUser(deps.Ctx, userID, tagPtr(1)) // Assuming tagPtr exists
				if createErr != nil {
					t.Fatalf("Failed to create user for test setup: %v", createErr)
				}
				if createResult.Success == nil {
					t.Fatalf("User creation failed in setup: %+v", createResult.Failure)
				}

				return deps.Ctx, userID, newRole
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected nil error for successful update, got: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Result contained non-nil Error for successful update, got: %v", result.Error)
				}
				if result.Success == nil {
					t.Fatal("Result contained nil Success payload, expected non-nil")
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload, got: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*userevents.UserRoleUpdateResultPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.UserRoleUpdateResultPayload")
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.Role != newRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", newRole, successPayload.Role)
				}

				// Verify the role was actually updated in the database
				getUserResult, getUserErr := deps.Service.GetUser(deps.Ctx, userID)
				if getUserErr != nil {
					t.Fatalf("Failed to retrieve user after update: %v", getUserErr)
				}
				if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayload).User.Role != newRole {
					t.Errorf("Database user role mismatch after update: expected %q, got %q", newRole, getUserResult.Success.(*userevents.GetUserResponsePayload).User.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - Invalid role",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, sharedtypes.UserRoleEnum) {
				userID := sharedtypes.DiscordID("123456789012345678")
				newRole := sharedtypes.UserRoleEnum("invalid_role")

				return deps.Ctx, userID, newRole
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for invalid role, got nil")
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

				failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserRoleUpdateFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				expectedFailureReason := "invalid role"
				if failurePayload.Reason != expectedFailureReason {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedFailureReason, failurePayload.Reason)
				}

				expectedWrappedErrMsg := "HandleUpdateUserRole operation failed: invalid role"
				if err.Error() != expectedWrappedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedWrappedErrMsg, err.Error())
				}
				expectedOriginalErrMsg := "invalid role"
				if result.Error.Error() != expectedOriginalErrMsg {
					t.Errorf("Result error message mismatch: expected %q, got %q", expectedOriginalErrMsg, result.Error.Error())
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - User does not exist",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, sharedtypes.UserRoleEnum) {
				userID := sharedtypes.DiscordID("99999999999999999")
				newRole := sharedtypes.UserRoleAdmin

				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}

				return deps.Ctx, userID, newRole
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for user not found, got nil")
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

				failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserRoleUpdateFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				expectedFailureReason := "user not found"
				if failurePayload.Reason != expectedFailureReason {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedFailureReason, failurePayload.Reason)
				}

				expectedWrappedErrMsg := "HandleUpdateUserRole operation failed: failed to update user role: user not found"
				if err.Error() != expectedWrappedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedWrappedErrMsg, err.Error())
				}
				expectedOriginalErr := userdb.ErrUserNotFound
				if !errors.Is(result.Error, expectedOriginalErr) {
					t.Errorf("Result error mismatch: expected error wrapping %v, got %v", expectedOriginalErr, result.Error)
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Nil context",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, sharedtypes.UserRoleEnum) {
				userID := sharedtypes.DiscordID("12345678901234567")
				newRole := sharedtypes.UserRoleAdmin
				return nil, userID, newRole
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum, result userservice.UserOperationResult, err error) {
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
			var newRole sharedtypes.UserRoleEnum
			var result userservice.UserOperationResult
			var err error

			// Setup dependencies based on skipCleanup flag
			if !tc.skipCleanup {
				// Assuming SetupTestUserService initializes deps.Service, deps.Ctx, deps.BunDB etc.
				currentDeps = SetupTestUserService(sharedCtx, sharedDB, t)

				// Clean database tables before the test
				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				// Run the test-specific setup function
				ctx, userID, newRole = tc.setupFn(t, currentDeps)
			} else {
				// For tests that skip cleanup or need a specific service setup (like nil context)
				// Initialize dependencies manually or with a specific helper
				currentDeps = TestDeps{
					Ctx: context.Background(), // Use a background context if test setup doesn't provide one
					Service: userservice.NewUserService(
						nil, // Replace with actual dependencies or mocks as needed
						nil,
						slog.New(slog.NewTextHandler(io.Discard, nil)),
						&usermetrics.NoOpMetrics{},
						noop.NewTracerProvider().Tracer("test"),
					),
					// Add other dependencies like BunDB if needed, potentially mocked
				}
				// Run the test-specific setup function with the manually created deps
				ctx, userID, newRole = tc.setupFn(t, currentDeps)
			}

			// Execute the function under test
			result, err = currentDeps.Service.UpdateUserRoleInDatabase(ctx, userID, newRole)

			// Validate the results using the test case's validateFn
			tc.validateFn(t, currentDeps, userID, newRole, result, err)

			// Optional cleanup after each test (if not skipped)
			if !tc.skipCleanup {
				if cleanupErr := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); cleanupErr != nil {
					t.Errorf("Failed to clean database tables after test %q: %v", tc.name, cleanupErr)
				}
			}
		})
	}
}
