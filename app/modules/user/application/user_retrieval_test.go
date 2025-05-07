package userservice

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		expectedOpResult UserOperationResult
		expectedErr      error
	}{
		{
			name: "Successfully retrieves user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID).
					Return(&userdbtypes.User{
						ID:     1,
						UserID: testUserID,
						Role:   sharedtypes.UserRoleAdmin,
					}, nil)
			},
			expectedOpResult: UserOperationResult{
				Success: &userevents.GetUserResponsePayload{
					User: &usertypes.UserData{
						ID:     1,
						UserID: testUserID,
						Role:   sharedtypes.UserRoleAdmin,
					},
				},
				Failure: nil,
				Error:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "User not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID).
					Return(nil, userdbtypes.ErrUserNotFound)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayload{
					UserID: testUserID,
					Reason: "user not found",
				},
				Error: errors.New("user not found"),
			},
			expectedErr: errors.New("user not found"), // Wrapper should return this error
		},
		{
			name: "Database error retrieving user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID).
					Return(nil, errors.New("database connection failed"))
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayload{
					UserID: testUserID,
					Reason: "failed to retrieve user from database",
				},
				Error: errors.New("database connection failed"),
			},
			expectedErr: errors.New("GetUser  operation failed: database connection failed"), // Wrapper wraps the original error
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

			gotResult, gotErr := s.GetUser(ctx, testUserID)

			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.GetUserResponsePayload)
				gotSuccess, ok := gotResult.Success.(*userevents.GetUserResponsePayload)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.GetUserResponsePayload, got %T", gotResult.Success)
				} else if gotSuccess == nil || gotSuccess.User == nil {
					t.Errorf("Expected non-nil success payload and user, got nil")
				} else if gotSuccess.User.UserID != expectedSuccess.User.UserID {
					t.Errorf("Mismatched UserID, got: %v, expected: %v", gotSuccess.User.UserID, expectedSuccess.User.UserID)
				} else if gotSuccess.User.Role != expectedSuccess.User.Role {
					t.Errorf("Mismatched Role, got: %v, expected: %v", gotSuccess.User.Role, expectedSuccess.User.Role)
				}
			} else if gotResult.Success != nil {
				t.Errorf("Unexpected success payload: %v", gotResult.Success)
			}

			if tt.expectedOpResult.Failure != nil {
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.GetUserFailedPayload)
				gotFailure, ok := gotResult.Failure.(*userevents.GetUserFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.GetUserFailedPayload, got %T", gotResult.Failure)
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

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		expectedOpResult UserOperationResult
		expectedErr      error
	}{
		{
			name: "Successfully retrieves user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleAdmin, nil)
			},
			expectedOpResult: UserOperationResult{
				Success: &userevents.GetUserRoleResponsePayload{
					UserID: testUserID,
					Role:   sharedtypes.UserRoleAdmin,
				},
				Failure: nil,
				Error:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Database error retrieving user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				dbErr := errors.New("failed to retrieve user role")
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleEnum(""), dbErr)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: testUserID,
					Reason: "failed to retrieve user role",
				},
				Error: errors.New("failed to retrieve user role"),
			},
			expectedErr: errors.New("GetUserRole operation failed: failed to retrieve user role"),
		},
		{
			name: "Retrieved invalid user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleEnum("InvalidRole"), nil)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: testUserID,
					Reason: "retrieved invalid user role",
				},
				Error: errors.New("invalid role in database"),
			},
			expectedErr: errors.New("GetUserRole operation failed: invalid role in database"),
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

			gotResult, gotErr := s.GetUserRole(ctx, testUserID)

			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.GetUserRoleResponsePayload)
				gotSuccess, ok := gotResult.Success.(*userevents.GetUserRoleResponsePayload)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.GetUserRoleResponsePayload, got %T", gotResult.Success)
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
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.GetUserRoleFailedPayload)
				gotFailure, ok := gotResult.Failure.(*userevents.GetUserRoleFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.GetUserRoleFailedPayload, got %T", gotResult.Failure)
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
