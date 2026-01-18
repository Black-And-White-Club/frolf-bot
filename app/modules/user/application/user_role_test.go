package userservice

import (
	"context"
	"errors" // Import fmt for error wrapping
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	results "github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserService_UpdateUserRoleInDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testRole := sharedtypes.UserRoleAdmin
	invalidRole := sharedtypes.UserRoleEnum("InvalidRole")
	dbErr := errors.New("database connection failed")

	mockDB := userdb.NewMockRepository(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockRepository)
		newRole          sharedtypes.UserRoleEnum
		expectedOpResult results.OperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper (which is the error from the inner serviceFunc)
	}{
		{
			name: "Successfully updates user role",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(nil)
			},
			newRole: testRole,
			expectedOpResult: results.SuccessResult(&userevents.UserRoleUpdateResultPayloadV1{
				UserID:  testUserID,
				Role:    testRole,
				Success: true,
				Reason:  "",
			}),
			expectedErr: nil,
		},
		{
			name: "Fails due to invalid role",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				// No database call expected for invalid role
			},
			newRole: invalidRole,
			expectedOpResult: results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{ // Expecting UserRoleUpdateResultPayloadV1
				UserID:  testUserID,
				Role:    invalidRole,
				Success: false,
				Reason:  "invalid role", // Reason set in the service code (Reason field)
			}),
			expectedErr: nil, // Mocked serviceWrapper returns nil error at top level for this case
		},
		{
			name: "Fails due to user not found",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(userdbtypes.ErrNotFound)
			},
			newRole: testRole,
			expectedOpResult: results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{ // Expecting UserRoleUpdateResultPayloadV1
				UserID:  testUserID,
				Role:    testRole,
				Success: false,
				Reason:  "user not found", // Reason set in the service code (Reason field)
			}),
			// Service now returns failure payload with Error populated but no top-level error
			expectedErr: nil,
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *userdb.MockRepository) {
				mockDB.EXPECT().
					UpdateUserRole(gomock.Any(), testUserID, testGuildID, testRole).
					Return(dbErr)
			},
			newRole: testRole,
			expectedOpResult: results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{ // Expecting UserRoleUpdateResultPayloadV1
				UserID:  testUserID,
				Role:    testRole,
				Success: false,
				Reason:  "failed to update user role", // Reason set in the service code (Reason field)
			}),
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

			gotResult, gotErr := s.UpdateUserRoleInDatabase(ctx, testGuildID, testUserID, tt.newRole)

			// Validate the returned UserOperationResult
			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.UserRoleUpdateResultPayloadV1)
				gotSuccess, ok := gotResult.Success.(*userevents.UserRoleUpdateResultPayloadV1)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.UserRoleUpdateResultPayloadV1, got %T", gotResult.Success)
				} else if gotSuccess == nil {
					t.Errorf("Expected non-nil success payload, got nil")
				} else if gotSuccess.UserID != expectedSuccess.UserID {
					t.Errorf("Mismatched UserID, got: %v, expected: %v", gotSuccess.UserID, expectedSuccess.UserID)
				} else if gotSuccess.Role != expectedSuccess.Role {
					t.Errorf("Mismatched Role, got: %v, expected: %v", gotSuccess.Role, expectedSuccess.Role)
				} else if gotSuccess.Success != expectedSuccess.Success {
					t.Errorf("Mismatched Success status, got: %v, expected: %v", gotSuccess.Success, expectedSuccess.Success)
				} else if gotSuccess.Reason != expectedSuccess.Reason {
					t.Errorf("Mismatched Reason message in success payload, got: %v, expected: %v", gotSuccess.Reason, expectedSuccess.Reason)
				}
			} else if gotResult.Success != nil {
				t.Errorf("Unexpected success payload: %v", gotResult.Success)
			}

			if tt.expectedOpResult.Failure != nil {
				// The service function returns UserRoleUpdateResultPayloadV1 for failures, not UserRoleUpdateFailedPayload
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.UserRoleUpdateResultPayloadV1)
				gotFailure, ok := gotResult.Failure.(*userevents.UserRoleUpdateResultPayloadV1)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.UserRoleUpdateResultPayloadV1, got %T", gotResult.Failure)
				} else if gotFailure == nil {
					t.Errorf("Expected non-nil failure payload, got nil")
				} else if gotFailure.Reason != expectedFailure.Reason { // Check the Reason field
					t.Errorf("Mismatched failure reason (Reason field), got: %v, expected: %v", gotFailure.Reason, expectedFailure.Reason)
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
