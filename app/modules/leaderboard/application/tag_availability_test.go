package leaderboardservice

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_CheckTagAvailability(t *testing.T) {
	tagNumber := sharedtypes.TagNumber(42)

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		userID         sharedtypes.DiscordID
		tagNumber      sharedtypes.TagNumber
		expectedResult *leaderboardevents.TagAvailabilityCheckResultPayload
		expectedFail   *leaderboardevents.TagAvailabilityCheckFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully checks available tag",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, tagNumber).Return(true, nil)
			},
			userID:    sharedtypes.DiscordID("test_user_id"),
			tagNumber: tagNumber,
			expectedResult: &leaderboardevents.TagAvailabilityCheckResultPayload{
				UserID:    sharedtypes.DiscordID("test_user_id"),
				TagNumber: &tagNumber,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Successfully checks unavailable tag",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, tagNumber).Return(false, nil)
			},
			userID:    sharedtypes.DiscordID("test_user_id"),
			tagNumber: tagNumber,
			expectedResult: &leaderboardevents.TagAvailabilityCheckResultPayload{
				UserID:    sharedtypes.DiscordID("test_user_id"),
				TagNumber: &tagNumber,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Database error when checking tag",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, tagNumber).Return(false, errors.New("database error"))
			},
			userID:         sharedtypes.DiscordID("test_user_id"),
			tagNumber:      tagNumber,
			expectedResult: nil,
			expectedFail: &leaderboardevents.TagAvailabilityCheckFailedPayload{
				UserID:    sharedtypes.DiscordID("test_user_id"),
				TagNumber: &tagNumber,
				Reason:    "failed to check tag availability",
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

			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &leaderboardmetrics.NoOpMetrics{}

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

			guildID := sharedtypes.GuildID("test-guild")
			got, got1, err := s.CheckTagAvailability(ctx, guildID, leaderboardevents.TagAvailabilityCheckRequestedPayload{
				UserID:    tt.userID,
				TagNumber: &tt.tagNumber,
			})

			if tt.expectedResult != nil {
				if got == nil {
					t.Errorf("Expected success payload, got nil")
				} else if got.UserID != tt.expectedResult.UserID || *got.TagNumber != *tt.expectedResult.TagNumber {
					t.Errorf("Mismatched success payload, got: %+v, expected: %+v", got, tt.expectedResult)
				}
			} else if got != nil {
				t.Errorf("Unexpected success payload: %v", got)
			}

			if tt.expectedFail != nil {
				if got1 == nil {
					t.Errorf("Expected failure payload, got nil")
				} else if got1.UserID != tt.expectedFail.UserID || *got1.TagNumber != *tt.expectedFail.TagNumber || got1.Reason != tt.expectedFail.Reason {
					t.Errorf("Mismatched failure payload, got: %+v, expected: %+v", got1, tt.expectedFail)
				}
			} else if got1 != nil {
				t.Errorf("Unexpected failure payload: %v", got1)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("Mismatched error reason, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
