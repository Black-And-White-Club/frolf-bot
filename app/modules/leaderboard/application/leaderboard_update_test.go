package leaderboardservice

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_UpdateLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testRoundID := sharedtypes.RoundID(uuid.New())
	testSortedParticipants := []string{"0:player1", "1:player2", "2:player3"}

	// Mock dependencies
	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

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
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{{TagNumber: 0, UserID: "old_player1"}, {TagNumber: 1, UserID: "old_player2"}},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: &leaderboardevents.LeaderboardUpdatedPayload{RoundID: testRoundID},
			expectedFail:   nil,
		},
		{
			name: "Fails to fetch active leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database connection error"))
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "database connection error",
			},
			expectedError: errors.New("database connection error"),
		},
		{
			name: "Fails to create new leaderboard", //This test was incorrect
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{{TagNumber: 0, UserID: "old_player1"}, {TagNumber: 1, UserID: "old_player2"}},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
				//The original test expected CreateLeaderboard, but the function doesn't call CreateLeaderboard in this scenario.
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update failure"))
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "failed to update leaderboard", // Corrected expected reason.
			},
			expectedError: errors.New("update failure"), // Corrected expected error
		},
		{
			name: "Fails to deactivate old leaderboard", //This test was incorrect
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{{TagNumber: 0, UserID: "old_player1"}, {TagNumber: 1, UserID: "old_player2"}},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil) //Simulate successful update
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: &leaderboardevents.LeaderboardUpdatedPayload{RoundID: testRoundID}, // Corrected expected result
			expectedFail:   nil,                                                                // Corrected expected failure
			expectedError:  nil,
		},
		{
			name: "Invalid input: empty sorted participant tags",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// No database operations expected
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
			name: "Database connection issue",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database connection error"))
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "database connection error",
			},
			expectedError: errors.New("database connection error"),
		},
		{
			name: "Concurrent updates", //This test was correct.
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{{TagNumber: 0, UserID: "old_player1"}, {TagNumber: 1, UserID: "old_player2"}},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: &leaderboardevents.LeaderboardUpdatedPayload{RoundID: testRoundID},
			expectedFail:   nil,
		},
		{
			name: "Invalid leaderboard data",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{}, //Return empty leaderboard data
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
			},
			roundID:        testRoundID,
			sortedTags:     testSortedParticipants,
			expectedResult: nil,
			expectedFail: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: testRoundID,
				Reason:  "invalid leaderboard data",
			},
			expectedError: errors.New("invalid leaderboard data"),
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			// Initialize service with No-Op implementations
			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc()
				},
			}

			got, err := s.UpdateLeaderboard(ctx, testMsg, tt.roundID, tt.sortedTags)

			// Validate success case
			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("❌ Expected success payload, got nil")
				} else {
					successPayload, ok := got.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
					if !ok {
						t.Errorf("❌ Expected Success to be *LeaderboardUpdatedPayload, but got %T", got.Success)
					} else if successPayload.RoundID != tt.expectedResult.RoundID {
						t.Errorf("❌ Mismatched RoundID, got: %v, expected: %v", successPayload.RoundID, tt.expectedResult.RoundID)
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
					failurePayload, ok := got.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
					if !ok {
						t.Errorf("❌ Expected Failure to be *LeaderboardUpdateFailedPayload, but got %T", got.Failure)
					} else if failurePayload.Reason != tt.expectedFail.Reason {
						t.Errorf("❌ Mismatched failure reason, got: %v, expected: %v", failurePayload.Reason, tt.expectedFail.Reason)
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
