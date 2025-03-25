package userservice

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testUserID := usertypes.DiscordID("12345678901234567")
	testTag := 42
	testTagPtr := &testTag

	// Mock dependencies
	mockDB := userdb.NewMockUserDB(ctrl)

	// Use No-Op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &usermetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockDBSetup    func(*userdb.MockUserDB)
		userID         usertypes.DiscordID
		tag            *int
		expectedResult *userevents.UserCreatedPayload
		expectedFail   *userevents.UserCreationFailedPayload
	}{
		{
			name: "Successfully creates a user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			userID: testUserID,
			tag:    testTagPtr,
			expectedResult: &userevents.UserCreatedPayload{
				UserID:    testUserID,
				TagNumber: testTagPtr,
			},
			expectedFail: nil,
		},
		{
			name: "Fails to create a user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("user already exists"))
			},
			userID:         testUserID,
			tag:            testTagPtr,
			expectedResult: nil,
			expectedFail: &userevents.UserCreationFailedPayload{
				UserID:    testUserID,
				TagNumber: testTagPtr,
				Reason:    "user already exists",
			},
		},
		{
			name: "With nil tag pointer",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			userID: testUserID,
			tag:    nil,
			expectedResult: &userevents.UserCreatedPayload{
				UserID:    testUserID,
				TagNumber: nil,
			},
			expectedFail: nil,
		},
		{
			name: "Fails due to unexpected database error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("database connection lost"))
			},
			userID:         testUserID,
			tag:            testTagPtr,
			expectedResult: nil,
			expectedFail: &userevents.UserCreationFailedPayload{
				UserID:    testUserID,
				TagNumber: testTagPtr,
				Reason:    "database connection lost",
			},
		},
		{
			name: "Fails due to empty Discord ID",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No expectations since the function should return early
			},
			userID:         "",
			tag:            testTagPtr,
			expectedResult: nil,
			expectedFail: &userevents.UserCreationFailedPayload{
				UserID:    "",
				TagNumber: testTagPtr,
				Reason:    "invalid Discord ID",
			},
		},
		{
			name: "Fails due to negative tag number",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No expectations since the function should return early
			},
			userID:         testUserID,
			tag:            &[]int{-1}[0],
			expectedResult: nil,
			expectedFail: &userevents.UserCreationFailedPayload{
				UserID:    testUserID,
				TagNumber: &[]int{-1}[0],
				Reason:    "tag number cannot be negative",
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
				serviceWrapper: func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc()
				},
			}

			gotSuccess, gotFailure, err := s.CreateUser(ctx, testMsg, tt.userID, tt.tag)

			// Validate success case
			if tt.expectedResult != nil {
				if gotSuccess == nil {
					t.Errorf("❌ Expected success payload, got nil")
				} else if gotSuccess.UserID != tt.expectedResult.UserID {
					t.Errorf("❌ Mismatched UserID, got: %v, expected: %v", gotSuccess.UserID, tt.expectedResult.UserID)
				}
			} else if gotSuccess != nil {
				t.Errorf("❌ Unexpected success payload: %v", gotSuccess)
			}

			// Validate failure case
			if tt.expectedFail != nil {
				if gotFailure == nil {
					t.Errorf("❌ Expected failure payload, got nil")
				} else if gotFailure.Reason != tt.expectedFail.Reason {
					t.Errorf("❌ Mismatched failure reason, got: %v, expected: %v", gotFailure.Reason, tt.expectedFail.Reason)
				}
			} else if gotFailure != nil {
				t.Errorf("❌ Unexpected failure payload: %v", gotFailure)
			}

			// Validate error presence
			if tt.expectedFail != nil {
				if err == nil {
					t.Errorf("❌ Expected an error but got nil")
				} else if err.Error() != tt.expectedFail.Reason {
					t.Errorf("❌ Mismatched error reason, got: %v, expected: %v", err.Error(), tt.expectedFail.Reason)
				}
			} else if err != nil {
				t.Errorf("❌ Unexpected error: %v", err)
			}
		})
	}
}
