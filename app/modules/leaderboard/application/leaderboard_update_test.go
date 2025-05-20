package leaderboardservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_UpdateLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)
	tag5 := sharedtypes.TagNumber(5)
	tag13 := sharedtypes.TagNumber(13)
	tag20 := sharedtypes.TagNumber(20)

	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())

	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		roundID        sharedtypes.RoundID
		sortedTags     []string
		expectedResult *leaderboardevents.LeaderboardUpdatedPayload
		expectedFail   *leaderboardevents.LeaderboardUpdateFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully updates leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				currentData := []leaderboardtypes.LeaderboardEntry{{TagNumber: tag2, UserID: "user2"}, {TagNumber: tag5, UserID: "user3"}, {TagNumber: tag13, UserID: "user1"}, {TagNumber: tag20, UserID: "user4"}}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: currentData,
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)

				// Expected data based on GenerateUpdatedLeaderboard logic with sortedTags ["13:user1", "5:user3", "2:user2"]
				// Participants: user1 (input tag 13), user3 (input tag 5), user2 (input tag 2).
				// Non-participant user4 keeps original tag 20.
				// Expected data before sorting: [{TagNumber: 13, UserID: "user1"}, {TagNumber: 5, UserID: "user3"}, {TagNumber: 2, UserID: "user2"}, {TagNumber: 20, UserID: "user4"}]
				expectedUpdatedData := []leaderboardtypes.LeaderboardEntry{
					{TagNumber: 2, UserID: "user2"},
					{TagNumber: 5, UserID: "user3"},
					{TagNumber: 13, UserID: "user1"},
					{TagNumber: 20, UserID: "user4"},
				}
				sortLeaderboardData(expectedUpdatedData)

				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), expectedUpdatedData, testRoundID).Return(nil)
			},
			roundID:        testRoundID,
			sortedTags:     []string{"13:user1", "5:user3", "2:user2"},
			expectedResult: &leaderboardevents.LeaderboardUpdatedPayload{RoundID: testRoundID},
			expectedFail:   nil,
			expectedError:  nil,
		},
		{
			name: "Fails to fetch active leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database connection error"))
			},
			roundID:        testRoundID,
			sortedTags:     []string{"1:user1"}, // Valid format, content doesn't matter for this error path
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "database connection error",
			},
			expectedError: errors.New("database connection error"),
		},
		{
			name: "Fails to update leaderboard in database",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Current leaderboard data with participants for this test
				currentData := []leaderboardtypes.LeaderboardEntry{{TagNumber: tag1, UserID: "player1"}, {TagNumber: tag2, UserID: "player2"}, {TagNumber: tag5, UserID: "player3"}}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: currentData,
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)

				// Participants in performance order: player1 (input tag 1), player3 (input tag 5), player2 (input tag 2)
				// Expected data before sorting: [{TagNumber: 1, UserID: "player1"}, {TagNumber: 5, UserID: "player3"}, {TagNumber: 2, UserID: "player2"}]
				expectedUpdatedData := []leaderboardtypes.LeaderboardEntry{
					{TagNumber: 1, UserID: "player1"},
					{TagNumber: 2, UserID: "player2"},
					{TagNumber: 5, UserID: "player3"},
				}
				sortLeaderboardData(expectedUpdatedData)

				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), expectedUpdatedData, testRoundID).Return(errors.New("update failure"))
			},
			roundID:        testRoundID,
			sortedTags:     []string{"1:player1", "5:player3", "2:player2"},
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "failed to update leaderboard in database",
			},
			expectedError: errors.New("failed to update leaderboard in database: update failure"),
		},
		{
			name: "Invalid input: empty sorted participant tags",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// No database operations expected for invalid input
			},
			roundID:        testRoundID,
			sortedTags:     []string{},
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "invalid input: empty sorted participant tags",
			},
			expectedError: errors.New("invalid input: empty sorted participant tags"),
		},
		{
			name: "GenerateUpdatedLeaderboard returns an error",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{{TagNumber: tag1, UserID: "old_player1"}},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
				// UpdateLeaderboard should NOT be called because GenerateUpdatedLeaderboard will return an error
			},
			roundID:        testRoundID,
			sortedTags:     []string{"1:user1", "invalid-tag-user"}, // Input that causes GenerateUpdatedLeaderboard to return an error
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "failed to generate updated leaderboard data: invalid sorted participant tag format: invalid-tag-user",
			},
			expectedError: errors.New("failed to generate updated leaderboard data: invalid sorted participant tag format: invalid-tag-user"),
		},
		{
			name: "Concurrent updates",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				currentData := []leaderboardtypes.LeaderboardEntry{{TagNumber: tag2, UserID: "user2"}, {TagNumber: tag5, UserID: "user3"}, {TagNumber: tag13, UserID: "user1"}, {TagNumber: tag20, UserID: "user4"}}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: currentData,
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)

				// Expected data based on GenerateUpdatedLeaderboard logic with sortedTags ["13:user1", "5:user3", "2:user2"]
				// Participants: user1 (input tag 13), user3 (input tag 5), user2 (input tag 2).
				// Non-participant user4 keeps original tag 20.
				// Expected data before sorting: [{TagNumber: 13, UserID: "user1"}, {TagNumber: 5, UserID: "user3"}, {TagNumber: 2, UserID: "user2"}, {TagNumber: 20, UserID: "user4"}]
				expectedUpdatedData := []leaderboardtypes.LeaderboardEntry{
					{TagNumber: 2, UserID: "user2"},
					{TagNumber: 5, UserID: "user3"},
					{TagNumber: 13, UserID: "user1"},
					{TagNumber: 20, UserID: "user4"},
				}
				sortLeaderboardData(expectedUpdatedData)

				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), expectedUpdatedData, testRoundID).Return(nil)
			},
			roundID:        testRoundID,
			sortedTags:     []string{"13:user1", "5:user3", "2:user2"},
			expectedResult: &leaderboardevents.LeaderboardUpdatedPayload{RoundID: testRoundID},
			expectedFail:   nil,
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			got, err := s.UpdateLeaderboard(ctx, tt.roundID, tt.sortedTags)

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload, got nil")
				} else {
					successPayload, ok := got.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
					if !ok {
						t.Errorf("Expected Success to be *LeaderboardUpdatedPayload, but got %T", got.Success)
					} else if successPayload.RoundID != tt.expectedResult.RoundID {
						t.Errorf("Mismatched RoundID in success payload, got: %v, expected: %v", successPayload.RoundID, tt.expectedResult.RoundID)
					}
				}
			} else if got.Success != nil {
				t.Errorf("Unexpected success payload: %v", got.Success)
			}

			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("Expected failure payload, got nil")
				} else {
					failurePayload, ok := got.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
					if !ok {
						t.Errorf("Expected Failure to be *LeaderboardUpdateFailedPayload, but got %T", got.Failure)
					} else {
						if failurePayload.Reason != tt.expectedFail.Reason {
							t.Errorf("Mismatched failure reason in failure payload, got: %v, expected: %v", failurePayload.Reason, tt.expectedFail.Reason)
						}
						if failurePayload.RoundID != tt.expectedFail.RoundID {
							t.Errorf("Mismatched RoundID in failure payload, got: %v, expected: %v", failurePayload.RoundID, tt.expectedFail.RoundID)
						}
					}
				}
			} else if got.Failure != nil {
				t.Errorf("Unexpected failure payload: %v", got.Failure)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				} else if !errors.Is(err, tt.expectedError) && !strings.Contains(err.Error(), tt.expectedError.Error()) {
					t.Errorf("Unexpected error: got %v, want %v", err, tt.expectedError)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
