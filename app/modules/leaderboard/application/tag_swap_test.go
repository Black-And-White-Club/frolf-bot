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
		payload        leaderboardevents.TagSwapRequestedPayload
		mockSetup      func(*leaderboarddb.MockLeaderboardDB) // Pass mockDB to setup
		expectedResult *LeaderboardOperationResult
		expectError    bool
	}{
		{
			name: "Successful tag swap",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard with both users present
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1) // Expect this call once

				// Mock the SwapTags call
				mdb.EXPECT().SwapTags(gomock.Any(), requestorID, targetID).Return(nil).Times(1) // Expect this call once
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
			name: "Cannot swap tag with self",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    requestorID, // Swap with self
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// No DB calls should happen in this case
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    requestorID,
					Reason:      "cannot swap tag with self",
				},
			},
			expectError: false,
		},
		{
			name: "No active leaderboard found",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard to return nil
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, nil).Times(1) // Expect this call once
				// No further DB calls should happen
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
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard to return an error
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("db error")).Times(1) // Expect this call once
				// No further DB calls should happen
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
			name: "Requestor does not have a tag",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: nonExistentID, // Requestor does not exist
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard with only the target user present
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1) // Expect this call once
				// No further DB calls should happen
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: nonExistentID,
					TargetID:    targetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name: "Target does not have a tag",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    nonExistentID, // Target does not exist
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard with only the requestor present
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1) // Expect this call once
				// No further DB calls should happen
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: requestorID,
					TargetID:    nonExistentID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name: "Neither user has a tag",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: nonExistentID, // Requestor does not exist
				TargetID:    targetID,      // Target does not exist (assuming targetID is not in the empty leaderboard)
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard with no users
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{}, // Empty leaderboard
				}, nil).Times(1) // Expect this call once
				// No further DB calls should happen
			},
			expectedResult: &LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: nonExistentID,
					TargetID:    targetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			},
			expectError: false,
		},
		{
			name: "Database error while swapping tags",
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB) {
				// Mock getting the active leaderboard with both users present
				mdb.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1) // Expect this call once

				// Mock the SwapTags call to return an error
				mdb.EXPECT().SwapTags(gomock.Any(), requestorID, targetID).Return(errors.New("db error")).Times(1) // Expect this call once
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
			// Reset mock expectations for each test case
			ctrl = gomock.NewController(t)
			mockDB = leaderboarddb.NewMockLeaderboardDB(ctrl)
			s.LeaderboardDB = mockDB // Update the service with the new mock
			defer ctrl.Finish()

			tt.mockSetup(mockDB) // Pass the mockDB to the setup function

			got, err := s.TagSwapRequested(ctx, tt.payload)

			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: got %v, expected error? %v", err, tt.expectError)
			}

			// Compare results based on whether success or failure is expected
			if tt.expectedResult.Success != nil {
				expectedSuccess, ok := tt.expectedResult.Success.(*leaderboardevents.TagSwapProcessedPayload)
				if !ok {
					t.Fatalf("Test setup error: expected Success payload of type *leaderboardevents.TagSwapProcessedPayload")
				}
				actualSuccess, ok := got.Success.(*leaderboardevents.TagSwapProcessedPayload)
				if !ok {
					t.Errorf("Expected success result, but got an unexpected type: %+v", got.Success)
					return
				}
				if actualSuccess.RequestorID != expectedSuccess.RequestorID ||
					actualSuccess.TargetID != expectedSuccess.TargetID {
					t.Errorf("Success result mismatch, got: %+v, expected: %+v", actualSuccess, expectedSuccess)
				}
				// Ensure Failure is nil
				if got.Failure != nil {
					t.Errorf("Expected success result, but Failure was not nil: %+v", got.Failure)
				}
			} else if tt.expectedResult.Failure != nil {
				expectedFailure, ok := tt.expectedResult.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Fatalf("Test setup error: expected Failure payload of type *leaderboardevents.TagSwapFailedPayload")
				}
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
				// Ensure Success is nil
				if got.Success != nil {
					t.Errorf("Expected failure result, but Success was not nil: %+v", got.Success)
				}
			} else {
				// Should not happen if expectedResult is properly set in test cases
				t.Fatalf("Test setup error: expectedResult must have either Success or Failure set")
			}
		})
	}
}
