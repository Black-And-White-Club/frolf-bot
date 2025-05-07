package userservice

import (
	"context"
	"errors"
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
		expectedErr      error
	}{
		{
			name: "Successfully updates user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(nil)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: &userevents.UserRoleUpdateResultPayload{
					UserID: testUserID,
					Role:   testRole,
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
				Failure: &userevents.UserRoleUpdateFailedPayload{
					UserID: testUserID,
					Reason: "invalid role",
				},
				Error: errors.New("invalid role"), // Error within the result
			},
			expectedErr: errors.New("invalid role"), // Top-level error from wrapper
		},
		{
			name: "Fails due to user not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(userdbtypes.ErrUserNotFound)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserRoleUpdateFailedPayload{
					UserID: testUserID,
					Reason: "user not found",
				},
				Error: userdbtypes.ErrUserNotFound, // Error within the result
			},
			expectedErr: errors.New("failed to update user role: user not found"), // Wrapper wraps ErrUserNotFound
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(dbErr)
			},
			newRole: testRole,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserRoleUpdateFailedPayload{
					UserID: testUserID,
					Reason: "failed to update user role",
				},
				Error: dbErr, // Error within the result
			},
			expectedErr: errors.New("failed to update user role: database connection failed"), // Wrapper wraps dbErr
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
				serviceWrapper: func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			gotResult, gotErr := s.UpdateUserRoleInDatabase(ctx, testUserID, tt.newRole)

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
				}
			} else if gotResult.Success != nil {
				t.Errorf("Unexpected success payload: %v", gotResult.Success)
			}

			if tt.expectedOpResult.Failure != nil {
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.UserRoleUpdateFailedPayload)
				gotFailure, ok := gotResult.Failure.(*userevents.UserRoleUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.UserRoleUpdateFailedPayload, got %T", gotResult.Failure)
				} else if gotFailure == nil {
					t.Errorf("Expected non-nil failure payload, got nil")
				} else if gotFailure.Reason != expectedFailure.Reason {
					t.Errorf("Mismatched failure reason, got: %v, expected: %v", gotFailure.Reason, expectedFailure.Reason)
				} else if gotFailure.UserID != expectedFailure.UserID {
					t.Errorf("Mismatched failure UserID, got: %v, expected: %v", gotFailure.UserID, expectedFailure.UserID)
				}
			} else if gotResult.Failure != nil {
				t.Errorf("Unexpected failure payload: %v", gotResult.Failure)
			}

			if tt.expectedOpResult.Error != nil {
				if gotResult.Error == nil {
					t.Errorf("Expected error in result, got nil")
				} else if gotResult.Error.Error() != tt.expectedOpResult.Error.Error() {
					t.Errorf("Mismatched result error reason, got: %v, expected: %v", gotResult.Error.Error(), tt.expectedOpResult.Error.Error())
				}
			} else if gotResult.Error != nil {
				t.Errorf("Unexpected error in result: %v", gotResult.Error)
			}

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
