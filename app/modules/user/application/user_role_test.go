package userservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_UpdateUserRoleInDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testUserID := usertypes.DiscordID("12345678901234567")
	testRole := usertypes.UserRoleAdmin

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
		newRole        usertypes.UserRoleEnum
		expectedResult *userevents.UserRoleUpdateResultPayload
		expectedFail   *userevents.UserRoleUpdateFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully updates user role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(nil)
			},
			newRole: testRole,
			expectedResult: &userevents.UserRoleUpdateResultPayload{
				UserID: testUserID,
				Role:   testRole,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails due to invalid role",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No database call expected for invalid role
			},
			newRole:        usertypes.UserRoleEnum("InvalidRole"),
			expectedResult: nil,
			expectedFail: &userevents.UserRoleUpdateFailedPayload{
				UserID: testUserID,
				Reason: "invalid role",
			},
			expectedError: errors.New("invalid role"),
		},
		{
			name: "Fails due to user not found",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(userdbtypes.ErrUserNotFound)
			},
			newRole:        testRole,
			expectedResult: nil,
			expectedFail: &userevents.UserRoleUpdateFailedPayload{
				UserID: testUserID,
				Reason: "user not found",
			},
			expectedError: errors.New("user not found"),
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testRole).
					Return(errors.New("database connection failed"))
			},
			newRole:        testRole,
			expectedResult: nil,
			expectedFail: &userevents.UserRoleUpdateFailedPayload{
				UserID: testUserID,
				Reason: "failed to update user role",
			},
			expectedError: errors.New("failed to update user role: database connection failed"),
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
				serviceWrapper: func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc()
				},
			}

			gotResult, gotFail, err := s.UpdateUserRoleInDatabase(ctx, testMsg, testUserID, tt.newRole)

			// Validate result
			if !reflect.DeepEqual(gotResult, tt.expectedResult) {
				t.Errorf("❌ Mismatched result, got: %v, expected: %v", gotResult, tt.expectedResult)
			}

			// Validate failure
			if !reflect.DeepEqual(gotFail, tt.expectedFail) {
				t.Errorf("❌ Mismatched failure, got: %v, expected: %v", gotFail, tt.expectedFail)
			}

			// Validate error
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("❌ Unexpected error: %v", err)
			} else if err != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("❌ Mismatched error message, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
			}
		})
	}
}
