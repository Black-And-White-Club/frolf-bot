package leaderboardservice

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagSwapRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
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

	requestorID := sharedtypes.DiscordID("user1")
	targetID := sharedtypes.DiscordID("user2")

	tests := []struct {
		name           string
		mockSetup      func()
		expectedResult *LeaderboardOperationResult
		expectError    bool
	}{
		{
			name: "Successful tag swap",
			mockSetup: func() {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID},
						{UserID: targetID},
					},
				}, nil)

				mockDB.EXPECT().SwapTags(gomock.Any(), requestorID, targetID).Return(nil)
			},
			expectedResult: &LeaderboardOperationResult{
				Success: &leaderboardevents.TagSwapProcessedPayload{
					RequestorID: requestorID,
					TargetID:    targetID,
				},
			},
			expectError: false,
		},

		{
			name: "No active leaderboard found",
			mockSetup: func() {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, nil)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "no active leaderboard found",
				},
			},
			expectError: false,
		},
		{
			name: "Database error while fetching leaderboard",
			mockSetup: func() {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("db error"))
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "db error",
				},
			},
			expectError: false,
		},
		{
			name: "One or both users do not have tags",
			mockSetup: func() {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID}, // Only requestor has a tag
					},
				}, nil)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name: "Database error while swapping tags",
			mockSetup: func() {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID},
						{UserID: targetID},
					},
				}, nil)

				mockDB.EXPECT().SwapTags(gomock.Any(), requestorID, targetID).Return(errors.New("db error"))
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "db error",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			payload := leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    targetID,
			}

			got, err := s.TagSwapRequested(ctx, payload)

			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: got %v, expected error? %v", err, tt.expectError)
			}

			if tt.expectedResult.Success != nil {
				expectedSuccess, _ := tt.expectedResult.Success.(*leaderboardevents.TagSwapProcessedPayload)
				actualSuccess, ok := got.Success.(*leaderboardevents.TagSwapProcessedPayload)
				if !ok {
					t.Errorf("Expected success result, but got an unexpected type: %+v", got.Success)
					return
				}
				if actualSuccess.RequestorID != expectedSuccess.RequestorID ||
					actualSuccess.TargetID != expectedSuccess.TargetID {
					t.Errorf("Success result mismatch, got: %+v, expected: %+v", actualSuccess, expectedSuccess)
				}
			} else if tt.expectedResult.Failure != nil {
				expectedFailure, _ := tt.expectedResult.Failure.(*leaderboardevents.TagSwapFailedPayload)
				actualFailure, ok := got.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Errorf("Expected failure result, but got an unexpected type: %+v", got.Failure)
					return
				}
				if actualFailure.RequestorID != expectedFailure.RequestorID ||
					actualFailure.TargetID != expectedFailure.TargetID ||
					actualFailure.Reason != expectedFailure.Reason {
					t.Errorf("Failure result mismatch, got: %+v, expected: %+v", actualFailure, expectedFailure)
				}
			}
		})
	}
}
