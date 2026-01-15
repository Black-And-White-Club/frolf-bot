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
	// Note: each subtest creates its own gomock Controller and defers Finish()
	// to avoid sharing a controller across subtests.
	var (
		mockDB *leaderboarddb.MockLeaderboardDB
	)
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
		targetTag      sharedtypes.TagNumber
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
			targetTag: sharedtypes.TagNumber(2),
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
				mdb.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), guildID, gomock.Any(), gomock.Any(), gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 2},
						{UserID: targetID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			expectedResult: &LeaderboardOperationResult{
				Leaderboard: leaderboardtypes.LeaderboardData{
					{UserID: requestorID, TagNumber: 2}, // After swap: requestor gets tag 2
					{UserID: targetID, TagNumber: 1},    // After swap: target gets tag 1
				},
				TagChanges: []TagChange{
					{
						GuildID: sharedtypes.GuildID("test-guild"),
						UserID:  requestorID,
						OldTag:  &[]sharedtypes.TagNumber{1}[0],
						NewTag:  &[]sharedtypes.TagNumber{2}[0],
						Reason:  sharedtypes.ServiceUpdateSourceTagSwap,
					},
					{
						GuildID: sharedtypes.GuildID("test-guild"),
						UserID:  targetID,
						OldTag:  &[]sharedtypes.TagNumber{2}[0],
						NewTag:  &[]sharedtypes.TagNumber{1}[0],
						Reason:  sharedtypes.ServiceUpdateSourceTagSwap,
					},
				},
				Err: nil,
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
				// Return a leaderboard where the requestor already holds the target tag
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			targetTag: sharedtypes.TagNumber(1),
			expectedResult: &LeaderboardOperationResult{
				Err: errors.New("cannot swap tag with self"),
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
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard).Times(1)
			},
			targetTag:      sharedtypes.TagNumber(0),
			expectedResult: nil,
			expectError:    true,
		},
		{
			name:    "Database error while fetching leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(nil, errors.New("db error")).Times(1)
			},
			targetTag:      sharedtypes.TagNumber(0),
			expectedResult: nil,
			expectError:    true,
		},
		{
			name:    "Requestor does not have a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: nonExistentID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockLeaderboardDB, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
			},
			targetTag: sharedtypes.TagNumber(2),
			expectedResult: &LeaderboardOperationResult{
				Err: errors.New("requesting user not on leaderboard"),
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
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			targetTag: sharedtypes.TagNumber(2),
			expectedResult: &LeaderboardOperationResult{
				Err: errors.New("target tag not currently assigned"),
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
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
				}, nil).Times(1)
			},
			targetTag: sharedtypes.TagNumber(2),
			expectedResult: &LeaderboardOperationResult{
				Err: errors.New("requesting user not on leaderboard"),
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
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
				mdb.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), guildID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).Times(1)
			},
			targetTag:      sharedtypes.TagNumber(2),
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for closure
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			s.LeaderboardDB = mockDB
			defer ctrl.Finish()

			tt.mockSetup(mockDB, tt.guildID)

			got, err := s.TagSwapRequested(ctx, tt.guildID, tt.payload.RequestorID, tt.targetTag)

			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: got %v, expected error? %v", err, tt.expectError)
			}

			if tt.expectedResult != nil {
				if tt.expectedResult.Err == nil {
					// Success case
					if got.Err != nil {
						t.Errorf("Expected success but got error: %v", got.Err)
						return
					}
					// Check leaderboard snapshot
					if len(got.Leaderboard) != len(tt.expectedResult.Leaderboard) {
						t.Errorf("Leaderboard length mismatch: got %d, expected %d", len(got.Leaderboard), len(tt.expectedResult.Leaderboard))
						return
					}
					for i, expected := range tt.expectedResult.Leaderboard {
						if got.Leaderboard[i].UserID != expected.UserID || got.Leaderboard[i].TagNumber != expected.TagNumber {
							t.Errorf("Leaderboard entry %d mismatch: got %+v, expected %+v", i, got.Leaderboard[i], expected)
						}
					}
					// Check tag changes
					if len(got.TagChanges) != len(tt.expectedResult.TagChanges) {
						t.Errorf("TagChanges length mismatch: got %d, expected %d", len(got.TagChanges), len(tt.expectedResult.TagChanges))
						return
					}
					for i, expected := range tt.expectedResult.TagChanges {
						if got.TagChanges[i].GuildID != expected.GuildID ||
							got.TagChanges[i].UserID != expected.UserID ||
							*got.TagChanges[i].OldTag != *expected.OldTag ||
							*got.TagChanges[i].NewTag != *expected.NewTag ||
							got.TagChanges[i].Reason != expected.Reason {
							t.Errorf("TagChange %d mismatch: got %+v, expected %+v", i, got.TagChanges[i], expected)
						}
					}
				} else {
					// Failure case
					if got.Err == nil {
						t.Errorf("Expected error but got success")
						return
					}
					if got.Err.Error() != tt.expectedResult.Err.Error() {
						t.Errorf("Error message mismatch: got %q, expected %q", got.Err.Error(), tt.expectedResult.Err.Error())
					}
				}
			}
		})
	}
}
