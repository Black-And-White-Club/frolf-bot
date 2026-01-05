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
	logger := loggerfrolfbot.NoOpLogger // Using a NoOp logger for tests
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{} // Using NoOp metrics for tests

	s := &LeaderboardService{
		LeaderboardDB: mockDB,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		// Use a serviceWrapper that simply executes the service function
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	ctx := context.Background()

	requestorID := sharedtypes.DiscordID("user1")
	targetID := sharedtypes.DiscordID("user2")
	nonExistentID := sharedtypes.DiscordID("user3")

	tests := []struct {
		name           string
		guildID        sharedtypes.GuildID
		payload        leaderboardevents.TagSwapRequestedPayloadV1
		mockSetup      func(*leaderboarddb.MockLeaderboardDB, sharedtypes.GuildID)
		expectedResult *LeaderboardOperationResult
		expectError    bool
	}{
		{
			name:    "Successful tag swap",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
				mdb.EXPECT().SwapTags(gomock.Any(), guildID, requestorID, targetID).Return(nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Success: &leaderboardevents.TagSwapProcessedPayloadV1{
					RequestorID: requestorID,
					TargetID:    targetID,
				},
			},
			expectError: false,
		},
		{
			name:    "Cannot swap tag with self",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    requestorID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				// No DB calls should happen in this case
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: requestorID,
					TargetID:    requestorID,
					Reason:      "cannot swap tag with self",
				},
			},
			expectError: false,
		},
		{
			name:    "No active leaderboard found",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(nil, nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "no active leaderboard found",
				},
			},
			expectError: false,
		},
		{
			name:    "Database error while fetching leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(nil, errors.New("db error")).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: requestorID,
					TargetID:    targetID,
					Reason:      "db error",
				},
			},
			expectError: false,
		},
		{
			name:    "Requestor does not have a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: nonExistentID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: nonExistentID,
					TargetID:    targetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name:    "Target does not have a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    nonExistentID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: requestorID,
					TargetID:    nonExistentID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name:    "Neither user has a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: nonExistentID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
				}, nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
					RequestorID: nonExistentID,
					TargetID:    targetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name:    "Database error while swapping tags",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
				mdb.EXPECT().SwapTags(gomock.Any(), guildID, requestorID, targetID).Return(errors.New("db error")).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayloadV1{
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
			ctrl = gomock.NewController(t)
			mockDB = leaderboarddb.NewMockLeaderboardDB(ctrl)
			s.LeaderboardDB = mockDB
			defer ctrl.Finish()

			tt.mockSetup(mockDB, tt.guildID)

			got, err := s.TagSwapRequested(ctx, tt.guildID, tt.payload)

			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: got %v, expected error? %v", err, tt.expectError)
			}

			if tt.expectedResult.Success != nil {
				expectedSuccess, ok := tt.expectedResult.Success.(*leaderboardevents.TagSwapProcessedPayloadV1)
				if !ok {
					t.Fatalf("Test setup error: expected Success payload of type *leaderboardevents.TagSwapProcessedPayloadV1")
				}
				actualSuccess, ok := got.Success.(*leaderboardevents.TagSwapProcessedPayloadV1)
				if !ok {
					t.Errorf("Expected success result, but got an unexpected type: %+v", got.Success)
					return
				}
				if actualSuccess.RequestorID != expectedSuccess.RequestorID ||
					actualSuccess.TargetID != expectedSuccess.TargetID {
					t.Errorf("Success result mismatch, got: %+v, expected: %+v", actualSuccess, expectedSuccess)
				}
				if got.Failure != nil {
					t.Errorf("Expected success result, but Failure was not nil: %+v", got.Failure)
				}
			} else if tt.expectedResult.Failure != nil {
				expectedFailure, ok := tt.expectedResult.Failure.(*leaderboardevents.TagSwapFailedPayloadV1)
				if !ok {
					t.Fatalf("Test setup error: expected Failure payload of type *leaderboardevents.TagSwapFailedPayloadV1")
				}
				actualFailure, ok := got.Failure.(*leaderboardevents.TagSwapFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result, but got an unexpected type: %+v", got.Failure)
					return
				}
				if actualFailure.RequestorID != expectedFailure.RequestorID ||
					actualFailure.TargetID != expectedFailure.TargetID ||
					actualFailure.Reason != expectedFailure.Reason {
					t.Errorf("Failure result mismatch, got: %+v, expected: %+v", actualFailure, expectedFailure)
				}
				if got.Success != nil {
					t.Errorf("Expected failure result, but Success was not nil: %+v", got.Success)
				}
			} else {
				t.Fatalf("Test setup error: expectedResult must have either Success or Failure set")
			}
		})
	}
}
