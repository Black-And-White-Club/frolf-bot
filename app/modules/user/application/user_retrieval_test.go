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
	testGuildID := sharedtypes.GuildID("98765432109876543")

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		expectedOpResult UserOperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper
	}{
		{
			name: "Successfully retrieves user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
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
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
					Return(nil, userdbtypes.ErrUserNotFound)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayload{
					UserID: testUserID,
					Reason: "user not found", // Reason set in the service code
				},
				Error: nil, // Service code returns nil error in result for user not found
			},
			expectedErr: nil, // Mocked serviceWrapper returns nil error at top level for user not found
		},
		{
			name: "Database error retrieving user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				dbErr := errors.New("database connection failed")
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
					Return(nil, dbErr)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayload{
					UserID: testUserID,
					Reason: "failed to retrieve user from database", // Reason set in the service code
				},
				Error: errors.New("database connection failed"), // Original DB error is returned in the result
			},
			expectedErr: errors.New("database connection failed"), // Mocked serviceWrapper returns the original DB error
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

			gotResult, gotErr := s.GetUser(ctx, testGuildID, testUserID)

			// Validate the returned UserOperationResult
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

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		expectedOpResult UserOperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper
	}{
		{
			name: "Successfully retrieves user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID, testGuildID).
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
					GetUserRole(gomock.Any(), testUserID, testGuildID).
					Return(sharedtypes.UserRoleEnum(""), dbErr)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: testUserID,
					Reason: "failed to retrieve user role from database", // Reason set in the service code
				},
				Error: errors.New("failed to retrieve user role"), // Original DB error is returned in the result
			},
			expectedErr: errors.New("failed to retrieve user role"), // Mocked serviceWrapper returns the original DB error
		},
		{
			name: "Retrieved invalid user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID, testGuildID).
					Return(sharedtypes.UserRoleEnum("InvalidRole"), nil)
			},
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: testUserID,
					Reason: "user found but has invalid role", // Reason set in the service code
				},
				Error: nil, // Service code returns nil error in result for invalid role
			},
			expectedErr: nil, // Mocked serviceWrapper returns nil error at top level for invalid role
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

			gotResult, gotErr := s.GetUserRole(ctx, testGuildID, testUserID)

			// Validate the returned UserOperationResult
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
