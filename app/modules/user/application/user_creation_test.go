package userservice

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTag := sharedtypes.TagNumber(42)
	negativeTag := sharedtypes.TagNumber(-1)

	mockDB := userdb.NewMockUserDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name             string
		mockDBSetup      func(*userdb.MockUserDB)
		userID           sharedtypes.DiscordID
		tag              *sharedtypes.TagNumber
		expectedOpResult UserOperationResult
		expectedErr      error // This should be the error returned by the mocked serviceWrapper
	}{
		{
			name: "Successfully creates a user",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			userID: testUserID,
			tag:    &testTag,
			expectedOpResult: UserOperationResult{
				Success: &userevents.UserCreatedPayload{
					UserID:    testUserID,
					TagNumber: &testTag,
				},
				Failure: nil,
				Error:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Fails to create a user due to DB error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// Simulate a DB error that translateDBError will convert to ErrUserAlreadyExists
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("SQLSTATE 23505: duplicate key value")) // Use a specific SQL error
			},
			userID: testUserID,
			tag:    &testTag,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    testUserID,
					TagNumber: &testTag,
					Reason:    ErrUserAlreadyExists.Error(), // Expected reason from translateDBError
				},
				Error: ErrUserAlreadyExists, // Expected error from translateDBError
			},
			expectedErr: ErrUserAlreadyExists, // The mocked serviceWrapper returns this error directly
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
			expectedOpResult: UserOperationResult{
				Success: &userevents.UserCreatedPayload{
					UserID:    testUserID,
					TagNumber: nil,
				},
				Failure: nil,
				Error:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Fails due to unexpected database error",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// Simulate an unexpected DB error
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("database connection lost"))
			},
			userID: testUserID,
			tag:    &testTag,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    testUserID,
					TagNumber: &testTag,
					Reason:    "database connection lost", // translateDBError returns original error
				},
				Error: errors.New("database connection lost"), // translateDBError returns original error
			},
			expectedErr: errors.New("database connection lost"), // The mocked serviceWrapper returns this error directly
		},
		{
			name: "Fails due to empty Discord ID",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No expectations since the function should return early
			},
			userID: "",
			tag:    &testTag,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    "",
					TagNumber: &testTag,
					Reason:    ErrInvalidDiscordID.Error(),
				},
				Error: ErrInvalidDiscordID,
			},
			expectedErr: ErrInvalidDiscordID,
		},
		{
			name: "Fails due to negative tag number",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No expectations since the function should return early
			},
			userID: testUserID,
			tag:    &negativeTag,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    testUserID,
					TagNumber: &negativeTag,
					Reason:    ErrNegativeTagNumber.Error(),
				},
				Error: ErrNegativeTagNumber,
			},
			expectedErr: ErrNegativeTagNumber,
		},
		{
			name: "Fails due to nil context",
			mockDBSetup: func(mockDB *userdb.MockUserDB) {
				// No expectations since the function should return early
			},
			userID: testUserID,
			tag:    &testTag,
			expectedOpResult: UserOperationResult{
				Success: nil,
				Failure: nil, // Nil context returns a failure payload with nil Failure field
				Error:   ErrNilContext,
			},
			expectedErr: ErrNilContext,
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

			// For the nil context test case, explicitly pass nil
			ctxArg := ctx
			if tt.name == "Fails due to nil context" {
				ctxArg = nil
			}

			gotResult, gotErr := s.CreateUser(ctxArg, testGuildID, tt.userID, tt.tag)

			// Validate the returned UserOperationResult
			if tt.expectedOpResult.Success != nil {
				expectedSuccess := tt.expectedOpResult.Success.(*userevents.UserCreatedPayload)
				gotSuccess, ok := gotResult.Success.(*userevents.UserCreatedPayload)
				if !ok {
					t.Errorf("Expected success payload of type *userevents.UserCreatedPayload, got %T", gotResult.Success)
				} else if gotSuccess.UserID != expectedSuccess.UserID {
					t.Errorf("Mismatched UserID, got: %v, expected: %v", gotSuccess.UserID, expectedSuccess.UserID)
				}
				// Compare tag numbers carefully due to pointers
				if (gotSuccess.TagNumber == nil && expectedSuccess.TagNumber != nil) ||
					(gotSuccess.TagNumber != nil && expectedSuccess.TagNumber == nil) ||
					(gotSuccess.TagNumber != nil && expectedSuccess.TagNumber != nil && *gotSuccess.TagNumber != *expectedSuccess.TagNumber) {
					t.Errorf("Mismatched TagNumber, got: %v, expected: %v", gotSuccess.TagNumber, expectedSuccess.TagNumber)
				}
			} else if gotResult.Success != nil {
				t.Errorf("Unexpected success payload: %v", gotResult.Success)
			}

			if tt.expectedOpResult.Failure != nil {
				expectedFailure := tt.expectedOpResult.Failure.(*userevents.UserCreationFailedPayload)
				gotFailure, ok := gotResult.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload of type *userevents.UserCreationFailedPayload, got %T", gotResult.Failure)
				} else if gotFailure.Reason != expectedFailure.Reason {
					t.Errorf("Mismatched failure reason, got: %v, expected: %v", gotFailure.Reason, expectedFailure.Reason)
				}
				if gotFailure.UserID != expectedFailure.UserID {
					t.Errorf("Mismatched failure UserID, got: %v, expected: %v", gotFailure.UserID, expectedFailure.UserID)
				}
				// Compare tag numbers carefully due to pointers
				if (gotFailure.TagNumber == nil && expectedFailure.TagNumber != nil) ||
					(gotFailure.TagNumber != nil && expectedFailure.TagNumber == nil) ||
					(gotFailure.TagNumber != nil && expectedFailure.TagNumber != nil && *gotFailure.TagNumber != *expectedFailure.TagNumber) {
					t.Errorf("Mismatched failure TagNumber, got: %v, expected: %v", gotFailure.TagNumber, expectedFailure.TagNumber)
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
