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
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID)
		validateFn       func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
		skipCleanup      bool
	}{
		{
			name: "Success - User exists",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)

				// Create a user first
				result, err := deps.Service.CreateUser(deps.Ctx, guildID, userID, tag, nil, nil)
				if err != nil {
					t.Fatalf("Failed to create user for test setup: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("User creation failed in setup: %+v", result.Failure)
				}

				return deps.Ctx, guildID, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
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

				expectedRole := sharedtypes.UserRoleEnum("User")
				if successPayload.User.Role != expectedRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", expectedRole, successPayload.User.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - User does not exist",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("99999999999999999")
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}
				return deps.Ctx, guildID, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected nil standard error for non-existent user, got: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Expected nil Error field in result for non-existent user, got: %v", result.Error)
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload for non-existent user, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload for non-existent user, expected non-nil")
				}
				failurePayload, ok := result.Failure.(*userevents.GetUserFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.GetUserFailedPayload, got %T", result.Failure)
				}
				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				expectedReason := "user not found"
				if failurePayload.Reason != expectedReason {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedReason, failurePayload.Reason)
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Empty Discord ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				return deps.Ctx, guildID, sharedtypes.DiscordID("")
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for empty Discord ID, got nil")
				}
				if result.Error != nil {
					t.Fatalf("Result contained unexpected Error: %v", result.Error)
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload, got: %+v", result.Failure)
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
			var guildID sharedtypes.GuildID
			var userID sharedtypes.DiscordID
			var result userservice.UserOperationResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(t)

				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				ctx, guildID, userID = tc.setupFn(t, currentDeps)
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
				ctx, guildID, userID = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.GetUser(ctx, guildID, userID)

			tc.validateFn(t, currentDeps, guildID, userID, result, err)
		})
	}
}

func TestGetUserRole(t *testing.T) {
	tests := []struct {
		name             string
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID)
		validateFn       func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
		skipCleanup      bool
	}{
		{
			name: "Success - Valid user role",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("12345678901234567")
				tag := tagPtr(42)
				result, err := deps.Service.CreateUser(deps.Ctx, guildID, userID, tag, nil, nil)
				if err != nil {
					t.Fatalf("Failed to create user for test setup: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("User creation failed in setup: %+v", result.Failure)
				}
				return deps.Ctx, guildID, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
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
				expectedRole := sharedtypes.UserRoleEnum("User")
				if successPayload.Role != expectedRole {
					t.Errorf("Success payload Role mismatch: expected %q, got %q", expectedRole, successPayload.Role)
				}
			},
			expectedSuccess: true,
			skipCleanup:     false,
		},
		{
			name: "Failure - User does not exist",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("99999999999999999")
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.BunDB); err != nil {
					t.Fatalf("Failed to clean database tables during setup: %v", err)
				}
				return deps.Ctx, guildID, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil { // This line is failing
					t.Fatalf("Expected nil standard error for non-existent user, got: %v", err) // This message will now show
				}
				if result.Error != nil { // This check was failing
					t.Fatalf("Expected nil Error field in result for non-existent user, got: %v", result.Error) // This message will now show
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload for non-existent user, got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatal("Result contained nil Failure payload for non-existent user, expected non-nil")
				}
				failurePayload, ok := result.Failure.(*userevents.GetUserRoleFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.GetUserRoleFailedPayload, got %T", result.Failure)
				}
				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				expectedReason := "user not found"
				if failurePayload.Reason != expectedReason {
					t.Errorf("Failure payload Reason mismatch: expected %q, got %q", expectedReason, failurePayload.Reason)
				}
			},
			expectedSuccess: false,
			skipCleanup:     false,
		},
		{
			name: "Failure - Nil context",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("12345678901234567")
				return nil, guildID, userID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
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
			var guildID sharedtypes.GuildID
			var userID sharedtypes.DiscordID
			var result userservice.UserOperationResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(t)
				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}
				ctx, guildID, userID = tc.setupFn(t, currentDeps)
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
				ctx, guildID, userID = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.GetUserRole(ctx, guildID, userID)

			tc.validateFn(t, currentDeps, guildID, userID, result, err)
		})
	}
}
