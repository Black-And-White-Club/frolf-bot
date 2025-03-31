package leaderboardservice

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_BatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)

	// Mock dependencies
	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		payload        leaderboardevents.BatchTagAssignmentRequestedPayload
		expectedResult LeaderboardOperationResult
		expectedError  error
	}{
		{
			name: "Successfully assigns tags",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), gomock.Any(), leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(nil)
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          "batch-1",
				RequestingUserID: "test_user_id",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedResult: LeaderboardOperationResult{
				Success: &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: "test_user_id",
					BatchID:          "batch-1",
					AssignmentCount:  2,
					Assignments: []leaderboardevents.TagAssignmentInfo{
						{UserID: "user1", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid tag assignment",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), gomock.Any(), leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(nil)
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          "batch-2",
				RequestingUserID: "test_user_id",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: -1}, // Invalid tag number
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedResult: LeaderboardOperationResult{
				Success: &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: "test_user_id",
					BatchID:          "batch-2",
					AssignmentCount:  1,
					Assignments: []leaderboardevents.TagAssignmentInfo{
						{UserID: "user2", TagNumber: 2}, // Only valid assignment should be returned
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error on batch assignment",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), gomock.Any(), leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(errors.New("database error"))
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          "batch-3",
				RequestingUserID: "test_user_id",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedResult: LeaderboardOperationResult{
				Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: "test_user_id",
					BatchID:          "batch-3",
					Reason:           "database error",
				},
			},
			expectedError: errors.New("database error"),
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

			got, err := s.BatchTagAssignmentRequested(ctx, testMsg, tt.payload)

			// Validate error presence
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				if got.Success == nil {
					t.Errorf("expected success result, got: nil")
				} else {
					successPayload, ok := got.Success.(*leaderboardevents.BatchTagAssignedPayload)
					if !ok {
						t.Errorf("expected result to be *BatchTagAssignedPayload, got: %T", got.Success)
					} else if successPayload.BatchID != tt.payload.BatchID {
						t.Errorf("expected BatchID: %v, got: %v", tt.payload.BatchID, successPayload.BatchID)
					}
				}
			}
		})
	}
}
