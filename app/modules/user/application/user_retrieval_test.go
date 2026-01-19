package userservice

import (
	"context"
	"errors"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	results "github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserService_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")

	mockDB := userdb.NewMockRepository(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockRepository)
		expectedOpResult results.OperationResult
		expectedErr      error
	}{
		{
			name: "Successfully retrieves user",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
					Return(&userdbtypes.UserWithMembership{
						User: &userdbtypes.User{
							ID:     1,
							UserID: testUserID,
						},
						Role:     sharedtypes.UserRoleAdmin,
						JoinedAt: time.Now(),
					}, nil)
			},
			expectedOpResult: results.SuccessResult(&userevents.GetUserResponsePayloadV1{
				User: &usertypes.UserData{
					ID:     1,
					UserID: testUserID,
					Role:   sharedtypes.UserRoleAdmin,
				},
			}),
			expectedErr: nil,
		},
		{
			name: "User not found",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
					Return(nil, userdbtypes.ErrNotFound)
			},
			expectedOpResult: results.FailureResult(&userevents.GetUserFailedPayloadV1{UserID: testUserID, Reason: "user not found"}),
			expectedErr:      nil,
		},
		{
			name: "Database error retrieving user",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				dbErr := errors.New("database connection failed")
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID, testGuildID).
					Return(nil, dbErr)
			},
			expectedOpResult: results.FailureResult(&userevents.GetUserFailedPayloadV1{UserID: testUserID, Reason: "failed to retrieve user from database"}),
			// Service now returns failure payload with Error populated but no top-level error
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &UserService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			gotResult, gotErr := s.GetUser(ctx, testGuildID, testUserID)

			// Validate the returned UserOperationResult
			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.GetUserResponsePayloadV1)
				gotSuccess, ok := gotResult.Success.(*userevents.GetUserResponsePayloadV1)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.GetUserResponsePayloadV1, got %T", gotResult.Success)
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
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.GetUserFailedPayloadV1)
				gotFailure, ok := gotResult.Failure.(*userevents.GetUserFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.GetUserFailedPayloadV1, got %T", gotResult.Failure)
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

func TestUserService_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")

	mockDB := userdb.NewMockRepository(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockRepository)
		expectedOpResult results.OperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper
	}{
		{
			name: "Successfully retrieves user role",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID, testGuildID).
					Return(sharedtypes.UserRoleAdmin, nil)
			},
			expectedOpResult: results.SuccessResult(&userevents.GetUserRoleResponsePayloadV1{
				UserID: testUserID,
				Role:   sharedtypes.UserRoleAdmin,
			}),
			expectedErr: nil,
		},
		{
			name: "Database error retrieving user role",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				dbErr := errors.New("failed to retrieve user role")
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID, testGuildID).
					Return(sharedtypes.UserRoleEnum(""), dbErr)
			},
			expectedOpResult: results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
				UserID: testUserID,
				Reason: "failed to retrieve user role from database", // Reason set in the service code
			}),
			// Service now returns failure payload with Error populated but no top-level error
			expectedErr: nil,
		},
		{
			name: "Retrieved invalid user role",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID, testGuildID).
					Return(sharedtypes.UserRoleEnum("InvalidRole"), nil)
			},
			expectedOpResult: results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
				UserID: testUserID,
				Reason: "user found but has invalid role", // Reason set in the service code
			}),
			expectedErr: nil, // Mocked serviceWrapper returns nil error at top level for invalid role
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &UserService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			gotResult, gotErr := s.GetUserRole(ctx, testGuildID, testUserID)

			// Validate the returned UserOperationResult
			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.GetUserRoleResponsePayloadV1)
				gotSuccess, ok := gotResult.Success.(*userevents.GetUserRoleResponsePayloadV1)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.GetUserRoleResponsePayloadV1, got %T", gotResult.Success)
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
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.GetUserRoleFailedPayloadV1)
				gotFailure, ok := gotResult.Failure.(*userevents.GetUserRoleFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.GetUserRoleFailedPayloadV1, got %T", gotResult.Failure)
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
