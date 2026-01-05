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
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("123456789012345678")
				newRole := sharedtypes.UserRoleAdmin

				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}

				// Create a user first
				createResult, createErr := deps.Service.CreateUser(deps.Ctx, guildID, userID, tagPtr(1), nil, nil)
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

				successPayload, ok := result.Success.(*userevents.UserRoleUpdateResultPayloadV1)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.UserRoleUpdateResultPayloadV1")
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.Role != newRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", newRole, successPayload.Role)
				}

				// Verify the role was actually updated in the database
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, getUserErr := deps.Service.GetUser(deps.Ctx, guildID, userID)
				if getUserErr != nil {
					t.Fatalf("Failed to retrieve user after update: %v", getUserErr)
				}
				if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayloadV1).User.Role != newRole {
					t.Errorf("Database user role mismatch after update: expected %q, got %q", newRole, getUserResult.Success.(*userevents.GetUserResponsePayloadV1).User.Role)
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
				if err != nil {
					t.Fatalf("Expected nil standard error for invalid role, got: %v", err)
				}
				if result.Error == nil {
					t.Fatal("Result contained nil Error, expected non-nil")
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}
				failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateResultPayloadV1)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserRoleUpdateResultPayloadV1, got %T", result.Failure)
				}
				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				if failurePayload.Role != newRole {
					t.Errorf("Failure payload Role mismatch: expected %q, got %q", newRole, failurePayload.Role)
				}
				if failurePayload.Success != false {
					t.Errorf("Failure payload Success mismatch: expected false, got %t", failurePayload.Success)
				}
				expectedFailureReasonString := "invalid role"
				if failurePayload.Reason != expectedFailureReasonString {
					t.Errorf("Failure payload Reason string mismatch: expected %q, got %q", expectedFailureReasonString, failurePayload.Reason)
				}
				expectedResultErrorString := "invalid role" // String comparison for result.Error
				if result.Error.Error() != expectedResultErrorString {
					t.Errorf("Result error string mismatch: expected %q, got %q", expectedResultErrorString, result.Error.Error())
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
				if err == nil { // Service returns non-nil err for DB errors
					t.Fatal("Expected error for user not found, got nil")
				}
				if result.Error == nil { // Service sets result.Error to the original DB error
					t.Fatal("Result contained nil Error, expected non-nil")
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload, expected non-nil")
				}
				failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateResultPayloadV1) // Service returns this type for failures
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserRoleUpdateResultPayloadV1, got %T", result.Failure)
				}
				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				if failurePayload.Role != newRole {
					t.Errorf("Failure payload Role mismatch: expected %q, got %q", newRole, failurePayload.Role)
				}
				if failurePayload.Success != false {
					t.Errorf("Failure payload Success mismatch: expected false, got %t", failurePayload.Success)
				}
				expectedFailureReasonString := "user not found" // Service puts reason in string Reason field
				if failurePayload.Reason != expectedFailureReasonString {
					t.Errorf("Failure payload Reason string mismatch: expected %q, got %q", expectedFailureReasonString, failurePayload.Reason)
				}
				expectedWrappedErrMsg := "HandleUpdateUserRole operation failed: failed to update user role: user not found"
				if err.Error() != expectedWrappedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedWrappedErrMsg, err.Error())
				}
				expectedOriginalErr := userdb.ErrUserNotFound // Service sets result.Error to the original DB error
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

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(t)
				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}
				ctx, userID, newRole = tc.setupFn(t, currentDeps)
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
				ctx, userID, newRole = tc.setupFn(t, currentDeps)
			}

			guildID := sharedtypes.GuildID("test-guild")
			result, err = currentDeps.Service.UpdateUserRoleInDatabase(ctx, guildID, userID, newRole)

			tc.validateFn(t, currentDeps, userID, newRole, result, err)

			if !tc.skipCleanup {
				if cleanupErr := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); cleanupErr != nil {
					t.Errorf("Failed to clean database tables after test %q: %v", tc.name, cleanupErr)
				}
			}
		})
	}
}
