package leaderboardservice

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagAssignmentRequested(t *testing.T) {
	tagNumber := sharedtypes.TagNumber(42)
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		userID         sharedtypes.DiscordID
		tagNumber      sharedtypes.TagNumber
		updateID       sharedtypes.RoundID
		expectedResult *leaderboardevents.TagAssignedPayload
		expectedFail   *leaderboardevents.TagAssignmentFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully assigns tag to user",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
				mockDB.EXPECT().AssignTag(gomock.Any(), gomock.Any(), gomock.Any(), leaderboarddbtypes.ServiceUpdateSourceCreateUser, gomock.Any()).Return(nil)
			},
			userID:    sharedtypes.DiscordID("test_user_id"),
			tagNumber: sharedtypes.TagNumber(42),
			updateID:  testRoundID,
			expectedResult: &leaderboardevents.TagAssignedPayload{
				UserID:    sharedtypes.DiscordID("test_user_id"),
				TagNumber: &tagNumber,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails to assign tag to user due to database error",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
				mockDB.EXPECT().AssignTag(gomock.Any(), gomock.Any(), gomock.Any(), leaderboarddbtypes.ServiceUpdateSourceCreateUser, gomock.Any()).Return(errors.New("database error"))
			},
			userID:         sharedtypes.DiscordID("test_user_id"),
			tagNumber:      sharedtypes.TagNumber(42),
			updateID:       testRoundID,
			expectedResult: nil,
			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
				UserID:     sharedtypes.DiscordID("test_user_id"),
				TagNumber:  &tagNumber,
				Source:     string(leaderboarddbtypes.ServiceUpdateSourceCreateUser),
				UpdateType: "",
				Reason:     "database error",
			},
			expectedError: errors.New("database error"),
		},
		{
			name: "Fails to assign tag to user due to invalid input",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
			},
			userID:         sharedtypes.DiscordID("test_user_id"),
			tagNumber:      -1,
			updateID:       testRoundID,
			expectedResult: nil,
			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
				UserID:     sharedtypes.DiscordID("test_user_id"),
				TagNumber:  nil,
				Source:     string(leaderboarddbtypes.ServiceUpdateSourceCreateUser),
				UpdateType: "",
				Reason:     "invalid input: invalid tag number",
			},
			expectedError: errors.New("invalid input: invalid tag number"),
		},
		{
			name: "Fails to assign tag to user due to GetActiveLeaderboard error",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database error"))
			},
			userID:         sharedtypes.DiscordID("test_user_id"),
			tagNumber:      sharedtypes.TagNumber(42),
			updateID:       testRoundID,
			expectedResult: nil,
			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
				UserID:     sharedtypes.DiscordID("test_user_id"),
				TagNumber:  &tagNumber,
				Source:     string(leaderboarddbtypes.ServiceUpdateSourceCreateUser),
				UpdateType: "",
				Reason:     "database error",
			},
			expectedError: errors.New("database error"),
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			// No-Op implementations for logging, metrics, and tracing
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &leaderboardmetrics.NoOpMetrics{}

			// Initialize service with No-Op implementations
			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			ctx := context.Background()

			got, err := s.TagAssignmentRequested(ctx, leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     tt.userID,
				TagNumber:  &tt.tagNumber,
				Source:     string(leaderboarddbtypes.ServiceUpdateSourceCreateUser),
				UpdateType: "",
			})

			// Validate success case
			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("❌ Expected success payload, got nil")
				} else {
					successPayload, ok := got.Success.(*leaderboardevents.TagAssignedPayload)
					if !ok {
						t.Errorf("❌ Expected Success to be *TagAssignedPayload, but got %T", got.Success)
					} else if successPayload.UserID != tt.expectedResult.UserID {
						t.Errorf("❌ Mismatched User ID, got: %v, expected: %v", successPayload.UserID, tt.expectedResult.UserID)
					} else if *successPayload.TagNumber != *tt.expectedResult.TagNumber {
						t.Errorf("❌ Mismatched Tag Number, got: %v, expected: %v", *successPayload.TagNumber, *tt.expectedResult.TagNumber)
					}
				}
			} else if got.Success != nil {
				t.Errorf("❌ Unexpected success payload: %v", got.Success)
			}

			// Validate failure case
			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("❌ Expected failure payload, got nil")
				} else {
					failurePayload, ok := got.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
					if !ok {
						t.Errorf("❌ Expected Failure to be *TagAssignmentFailedPayload, but got %T", got.Failure)
					} else if failurePayload.UserID != tt.expectedFail.UserID {
						t.Errorf("❌ Mismatched User ID, got: %v, expected: %v", failurePayload.UserID, tt.expectedFail.UserID)
					} else if (failurePayload.TagNumber == nil && tt.expectedFail.TagNumber != nil) ||
						(failurePayload.TagNumber != nil && tt.expectedFail.TagNumber == nil) ||
						(failurePayload.TagNumber != nil && tt.expectedFail.TagNumber != nil && *failurePayload.TagNumber != *tt.expectedFail.TagNumber) {
						t.Errorf("❌ Mismatched Tag Number, got: %v, expected: %v", failurePayload.TagNumber, tt.expectedFail.TagNumber)
					} else if failurePayload.Source != tt.expectedFail.Source {
						t.Errorf("❌ Mismatched Source, got: %v, expected: %v", failurePayload.Source, tt.expectedFail.Source)
					} else if failurePayload.UpdateType != tt.expectedFail.UpdateType {
						t.Errorf("❌ Mismatched Update Type, got: %v, expected: %v", failurePayload.UpdateType, tt.expectedFail.UpdateType)
					} else if failurePayload.Reason != tt.expectedFail.Reason {
						t.Errorf("❌ Mismatched Reason, got: %v, expected: %v", failurePayload.Reason, tt.expectedFail.Reason)
					}
				}
			} else if got.Failure != nil {
				t.Errorf("❌ Unexpected failure payload: %v", got.Failure)
			}

			// Validate error presence
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("❌ Expected an error but got nil")
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("❌ Mismatched error reason, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
				}
			} else if err != nil {
				t.Errorf("❌ Unexpected error: %v", err)
			}
		})
	}
}
