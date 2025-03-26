package userservice

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testUserID := sharedtypes.DiscordID("12345678901234567")

	// Mock dependencies
	mockDB := userdb.NewMockUserDB(ctrl)

	// Use No-Op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &usermetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	// Define test cases
	tests := []struct {
		name           string
		mockDBSetup    func(*userdb.MockUserDB)
		expectedResult *userevents.GetUserResponsePayload
		expectedFail   *userevents.GetUserFailedPayload
		expectedError  error
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
			expectedResult: &userevents.GetUserResponsePayload{
				User: &usertypes.UserData{
					ID:     1,
					UserID: testUserID,
					Role:   sharedtypes.UserRoleAdmin,
				},
			},
			expectedFail: nil,
		},
		{
			name: "User not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID).
					Return(nil, userdbtypes.ErrUserNotFound)
			},
			expectedResult: nil,
			expectedFail: &userevents.GetUserFailedPayload{
				UserID: testUserID,
				Reason: "user not found",
			},
			expectedError: errors.New("user not found"),
		},
		{
			name: "Database error retrieving user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserByUserID(gomock.Any(), testUserID).
					Return(nil, errors.New("database connection failed"))
			},
			expectedResult: nil,
			expectedFail: &userevents.GetUserFailedPayload{
				UserID: testUserID,
				Reason: "failed to retrieve user from database",
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			// Initialize service with No-Op implementations
			s := &UserServiceImpl{
				UserDB:  mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(msg *message.Message, operationName string, userID sharedtypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc()
				},
			}

			gotSuccess, gotFailure, err := s.GetUser(ctx, testMsg, testUserID)

			// Validate success
			if tt.expectedResult != nil {
				if gotSuccess == nil {
					t.Errorf("❌ Expected success payload, got nil")
				} else if gotSuccess.User.UserID != tt.expectedResult.User.UserID {
					t.Errorf("❌ Mismatched UserID, got: %v, expected: %v", gotSuccess.User.UserID, tt.expectedResult.User.UserID)
				}
			}

			// Validate failure
			if tt.expectedFail != nil {
				if gotFailure == nil {
					t.Errorf("❌ Expected failure payload, got nil")
				} else if gotFailure.Reason != tt.expectedFail.Reason {
					t.Errorf("❌ Mismatched failure reason, got: %v, expected: %v", gotFailure.Reason, tt.expectedFail.Reason)
				}
			}

			// Validate error presence
			if (err != nil) != (tt.expectedFail != nil) {
				t.Errorf("❌ Unexpected error: %v", err)
			}
		})
	}
}

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testUserID := sharedtypes.DiscordID("12345678901234567")

	// Mock dependencies
	mockDB := userdb.NewMockUserDB(ctrl)

	// Use No-Op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &usermetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	// Define test cases
	tests := []struct {
		name           string
		mockDBSetup    func(*userdb.MockUserDB)
		expectedResult *userevents.GetUserRoleResponsePayload
		expectedFail   *userevents.GetUserRoleFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleAdmin, nil)
			},
			expectedResult: &userevents.GetUserRoleResponsePayload{
				UserID: testUserID,
				Role:   sharedtypes.UserRoleAdmin,
			},
			expectedFail: nil,
		},
		{
			name: "User role not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleEnum(""), errors.New("failed to retrieve user role"))
			},
			expectedResult: nil,
			expectedFail: &userevents.GetUserRoleFailedPayload{
				UserID: testUserID,
				Reason: "failed to retrieve user role",
			},
			expectedError: errors.New("failed to retrieve user role"),
		},
		{
			name: "Retrieved invalid user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					GetUserRole(gomock.Any(), testUserID).
					Return(sharedtypes.UserRoleEnum("InvalidRole"), nil)
			},
			expectedResult: nil,
			expectedFail: &userevents.GetUserRoleFailedPayload{
				UserID: testUserID,
				Reason: "retrieved invalid user role",
			},
			expectedError: errors.New("invalid role in database"),
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			// Initialize service with No-Op implementations
			s := &UserServiceImpl{
				UserDB:  mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(msg *message.Message, operationName string, userID sharedtypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc()
				},
			}

			gotSuccess, gotFailure, err := s.GetUserRole(ctx, testMsg, testUserID)

			// Validate success
			if tt.expectedResult != nil {
				if gotSuccess == nil {
					t.Errorf("❌ Expected success payload, got nil")
				} else if gotSuccess.Role != tt.expectedResult.Role {
					t.Errorf("❌ Mismatched role, got: %v, expected: %v", gotSuccess.Role, tt.expectedResult.Role)
				}
			}

			// Validate failure
			if tt.expectedFail != nil {
				if gotFailure == nil {
					t.Errorf("❌ Expected failure payload, got nil")
				} else if gotFailure.Reason != tt.expectedFail.Reason {
					t.Errorf("❌ Mismatched failure reason, got: %v, expected: %v", gotFailure.Reason, tt.expectedFail.Reason)
				}
			}

			// Validate error presence
			if (err != nil) != (tt.expectedFail != nil) {
				t.Errorf("❌ Unexpected error: %v", err)
			}
		})
	}
}
