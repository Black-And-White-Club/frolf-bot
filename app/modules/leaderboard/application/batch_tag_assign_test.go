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
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_BatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

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
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			got, err := s.BatchTagAssignmentRequested(ctx, tt.payload)

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
