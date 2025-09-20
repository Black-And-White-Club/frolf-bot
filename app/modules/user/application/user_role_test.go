package userservice

import (
	"context"
	"errors"
	"fmt" // Import fmt for error wrapping
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_UpdateUserRoleInDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testRole := sharedtypes.UserRoleAdmin
	invalidRole := sharedtypes.UserRoleEnum("InvalidRole")
	dbErr := errors.New("database connection failed")

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		newRole          sharedtypes.UserRoleEnum
		expectedOpResult UserOperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper (which is the error from the inner serviceFunc)
	}{
		{
			name: "Successfully updates user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(nil)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: &userevents.UserRoleUpdateResultPayload{
					UserID:  testUserID,
					Role:    testRole,
					Success: true,
					Error:   "",
				},
				Failure: nil,
				Error:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Fails due to invalid role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No database call expected for invalid role
			},
			newRole: invalidRole,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserRoleUpdateResultPayload{ // Expecting UserRoleUpdateResultPayload
					UserID:  testUserID,
					Role:    invalidRole,
					Success: false,
					Error:   "invalid role", // Reason set in the service code (Error field)
				},
				Error: errors.New("invalid role"), // Error within the result
			},
			expectedErr: nil, // Mocked serviceWrapper returns nil error at top level for this case
		},
		{
			name: "Fails due to user not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(userdbtypes.ErrUserNotFound)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserRoleUpdateResultPayload{ // Expecting UserRoleUpdateResultPayload
					UserID:  testUserID,
					Role:    testRole,
					Success: false,
					Error:   "user not found", // Reason set in the service code (Error field)
				},
				Error: userdbtypes.ErrUserNotFound, // Error within the result
			},
			// Expect the wrapped error returned by the service function
			expectedErr: fmt.Errorf("failed to update user role: %w", userdbtypes.ErrUserNotFound),
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(dbErr)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserRoleUpdateResultPayload{ // Expecting UserRoleUpdateResultPayload
					UserID:  testUserID,
					Role:    testRole,
					Success: false,
					Error:   "failed to update user role", // Reason set in the service code (Error field)
				},
				Error: dbErr, // Error within the result
			},
			// Expect the wrapped error returned by the service function
			expectedErr: fmt.Errorf("failed to update user role: %w", dbErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &UserServiceImpl{
				UserDB:  mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				// Mock the serviceWrapper to call the provided function directly in tests
				serviceWrapper: func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					// In the test wrapper, just call the actual service function and return its result
					// We skip the real wrapper's tracing, logging, metrics, panic recovery for simplicity
					// as these are concerns of the wrapper itself, not the service method logic being tested.
					return serviceFunc(ctx)
				},
			}

			gotResult, gotErr := s.UpdateUserRoleInDatabase(ctx, testGuildID, testUserID, tt.newRole)

			// Validate the returned UserOperationResult
			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.UserRoleUpdateResultPayload)
				gotSuccess, ok := gotResult.Success.(*userevents.UserRoleUpdateResultPayload)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.UserRoleUpdateResultPayload, got %T", gotResult.Success)
				} else if gotSuccess == nil {
					t.Errorf("Expected non-nil success payload, got nil")
				} else if gotSuccess.UserID != expectedSuccess.UserID {
					t.Errorf("Mismatched UserID, got: %v, expected: %v", gotSuccess.UserID, expectedSuccess.UserID)
				} else if gotSuccess.Role != expectedSuccess.Role {
					t.Errorf("Mismatched Role, got: %v, expected: %v", gotSuccess.Role, expectedSuccess.Role)
				} else if gotSuccess.Success != expectedSuccess.Success {
					t.Errorf("Mismatched Success status, got: %v, expected: %v", gotSuccess.Success, expectedSuccess.Success)
				} else if gotSuccess.Error != expectedSuccess.Error {
					t.Errorf("Mismatched Error message in success payload, got: %v, expected: %v", gotSuccess.Error, expectedSuccess.Error)
				}
			} else if gotResult.Success != nil {
				t.Errorf("Unexpected success payload: %v", gotResult.Success)
			}

			if tt.expectedOpResult.Failure != nil {
				// The service function returns UserRoleUpdateResultPayload for failures, not UserRoleUpdateFailedPayload
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.UserRoleUpdateResultPayload)
				gotFailure, ok := gotResult.Failure.(*userevents.UserRoleUpdateResultPayload)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.UserRoleUpdateResultPayload, got %T", gotResult.Failure)
				} else if gotFailure == nil {
					t.Errorf("Expected non-nil failure payload, got nil")
				} else if gotFailure.Error != expectedFailure.Error { // Check the Error field
					t.Errorf("Mismatched failure reason (Error field), got: %v, expected: %v", gotFailure.Error, expectedFailure.Error)
				} else if gotFailure.UserID != expectedFailure.UserID {
					t.Errorf("Mismatched failure UserID, got: %v, expected: %v", gotFailure.UserID, expectedFailure.UserID)
				} else if gotFailure.Success != expectedFailure.Success {
					t.Errorf("Mismatched Success status in failure payload, got: %v, expected: %v", gotFailure.Success, expectedFailure.Success)
				} else if gotFailure.Role != expectedFailure.Role {
					t.Errorf("Mismatched Role in failure payload, got: %v, expected: %v", gotFailure.Role, expectedFailure.Role)
				}

			} else if gotResult.Failure != nil {
				t.Errorf("Unexpected failure payload: %v", gotResult.Failure)
			}

			// Validate the Error field within the UserOperationResult
			if tt.expectedOpResult.Error != nil {
				if gotResult.Error == nil {
					t.Errorf("Expected error in result, got nil")
				} else if gotResult.Error.Error() != tt.expectedOpResult.Error.Error() {
					t.Errorf("Mismatched result error reason, got: %v, expected: %v", gotResult.Error.Error(), tt.expectedOpResult.Error.Error())
				}
			} else if gotResult.Error != nil {
				t.Errorf("Unexpected error in result: %v", gotResult.Error)
			}

			// Validate the top-level returned error
			if tt.expectedErr != nil {
				if gotErr == nil {
					t.Errorf("Expected a top-level error but got nil")
				} else if gotErr.Error() != tt.expectedErr.Error() {
					t.Errorf("Mismatched top-level error reason, got: %v, expected: %v", gotErr.Error(), tt.expectedErr.Error())
				}
			} else if gotErr != nil {
				t.Errorf("Unexpected top-level error: %v", gotErr)
			}
		})
	}
}
