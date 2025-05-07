// integration_tests/modules/user/user_creation_test.go
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

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name             string
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.DiscordID, *sharedtypes.TagNumber)
		validateFn       func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error)
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
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
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

				successPayload, ok := result.Success.(*userevents.UserCreatedPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.UserCreatedPayload")
				}

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, userID)
				if dbErr != nil {
					t.Fatalf("Failed to retrieve user %q from DB: %v", userID, dbErr)
				}
				if retrievedUser == nil {
					t.Fatalf("User %q not found in database after creation", userID)
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.TagNumber == nil {
					t.Errorf("Success payload TagNumber is unexpectedly nil")
				} else if *successPayload.TagNumber != 42 {
					t.Errorf("Success payload TagNumber mismatch: expected %d, got %d", 42, *successPayload.TagNumber)
				}

				if retrievedUser.UserID != userID {
					t.Errorf("Retrieved user UserID mismatch: expected %q, got %q", userID, retrievedUser.UserID)
				}

				expectedRole := sharedtypes.UserRoleEnum("Rattler")
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

				failurePayload, ok := result.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserCreationFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected empty string, got %q", failurePayload.UserID)
				}
				if failurePayload.TagNumber == nil {
					t.Errorf("Failure payload TagNumber is unexpectedly nil")
				} else if *failurePayload.TagNumber != 42 {
					t.Errorf("Failure payload TagNumber mismatch: expected %d, got %d", 42, *failurePayload.TagNumber)
				}

				expectedErrMsg := "invalid Discord ID"
				if err.Error() != expectedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedErrMsg, err.Error())
				}
				if result.Error.Error() != expectedErrMsg {
					t.Errorf("Result error message mismatch: expected %q, got %q", expectedErrMsg, result.Error.Error())
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
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for negative tag number, got nil")
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

				failurePayload, ok := result.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserCreationFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
				}
				if failurePayload.TagNumber == nil {
					t.Errorf("Failure payload TagNumber is unexpectedly nil")
				} else if *failurePayload.TagNumber != -5 {
					t.Errorf("Failure payload TagNumber mismatch: expected %d, got %d", -5, *failurePayload.TagNumber)
				}

				expectedErrMsg := "tag number cannot be negative"
				if err.Error() != expectedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedErrMsg, err.Error())
				}
				if result.Error.Error() != expectedErrMsg {
					t.Errorf("Result error message mismatch: expected %q, got %q", expectedErrMsg, result.Error.Error())
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
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
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

				successPayload, ok := result.Success.(*userevents.UserCreatedPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.UserCreatedPayload")
				}

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, userID)
				if dbErr != nil {
					t.Fatalf("Failed to retrieve user %q from DB: %v", userID, dbErr)
				}
				if retrievedUser == nil {
					t.Fatalf("User %q not found in database after creation", userID)
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.TagNumber != nil {
					t.Errorf("Success payload TagNumber should be nil, got %d", *successPayload.TagNumber)
				}

				if retrievedUser.UserID != userID {
					t.Errorf("Retrieved user UserID mismatch: expected %q, got %q", userID, retrievedUser.UserID)
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
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err == nil {
					t.Fatal("Expected error for nil context, got nil")
				}
				if result.Error == nil {
					t.Fatalf("Result contained nil Error, expected non-nil: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Result contained non-nil Success payload, got: %+v", result.Success)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload, got: %+v", result.Failure)
				}

				expectedErrMsg := "context cannot be nil"
				if err.Error() != expectedErrMsg {
					t.Errorf("Returned error message mismatch: expected %q, got %q", expectedErrMsg, err.Error())
				}
				if result.Error.Error() != expectedErrMsg {
					t.Errorf("Result error message mismatch: expected %q, got %q", expectedErrMsg, result.Error.Error())
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
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				if err != nil {
					t.Fatalf("CreateUser returned unexpected error: %v", err)
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

				successPayload, ok := result.Success.(*userevents.UserCreatedPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *userevents.UserCreatedPayload")
				}

				retrievedUser, dbErr := deps.DB.GetUserByUserID(deps.Ctx, userID)
				if dbErr != nil {
					t.Fatalf("Failed to retrieve user %q from DB: %v", userID, dbErr)
				}
				if retrievedUser == nil {
					t.Fatalf("User with large tag number not found in database")
				}

				if successPayload.UserID != userID {
					t.Errorf("Success payload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.TagNumber == nil {
					t.Errorf("Success payload TagNumber is unexpectedly nil")
				} else if *successPayload.TagNumber != 999999 {
					t.Errorf("Success payload TagNumber mismatch: expected %d, got %d", 999999, *successPayload.TagNumber)
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

				// Call CreateUser for setup and check its result
				result, err := deps.Service.CreateUser(deps.Ctx, userID, tag)
				if err != nil && result.Failure == nil { // If error exists but no failure payload, something is wrong
					t.Fatalf("Failed to setup test by creating initial user: %v (Result: %+v)", err, result)
				}
				// Check if the initial creation succeeded as expected
				if result.Success == nil {
					t.Fatalf("Setup failed to create initial user successfully: %+v", result.Failure)
				}
				if err != nil {
					t.Fatalf("Setup created user but returned unexpected error: %v", err)
				}

				// Return the same userID to attempt creation again in the test logic
				return deps.Ctx, userID, tag
			},
			validateFn: func(t *testing.T, deps TestDeps, userID sharedtypes.DiscordID, result userservice.UserOperationResult, err error) {
				// We expect the second creation attempt to fail
				if err == nil {
					t.Fatal("Expected error when creating duplicate user, got nil")
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

				failurePayload, ok := result.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type *userevents.UserCreationFailedPayload")
				}

				if failurePayload.UserID != userID {
					t.Errorf("Failure payload UserID mismatch: expected %q, got %q", userID, failurePayload.UserID)
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
			var result userservice.UserOperationResult
			var err error

			if !tc.skipCleanup {
				currentDeps = SetupTestUserService(sharedCtx, sharedDB, t)

				if err := testutils.CleanUserIntegrationTables(currentDeps.Ctx, currentDeps.BunDB); err != nil {
					t.Fatalf("Failed to clean database before test %q: %v", tc.name, err)
				}

				ctx, userID, tag = tc.setupFn(t, currentDeps)

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
				ctx, userID, tag = tc.setupFn(t, currentDeps)
			}

			result, err = currentDeps.Service.CreateUser(ctx, userID, tag)

			tc.validateFn(t, currentDeps, userID, result, err)
		})
	}
}
