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
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagSwapRequested(t *testing.T) {
	// Note: each subtest creates its own gomock Controller and defers Finish()
	// to avoid sharing a controller across subtests.
	// mockDB will be created per-subtest
	logger := loggerfrolfbot.NoOpLogger // Using a NoOp logger for tests
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{} // Using NoOp metrics for tests

	s := &LeaderboardService{
		repo:    nil,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}

	ctx := context.Background()

	requestorID := sharedtypes.DiscordID("user1")
	targetID := sharedtypes.DiscordID("user2")
	nonExistentID := sharedtypes.DiscordID("user3")

	tests := []struct {
		name               string
		guildID            sharedtypes.GuildID
		payload            leaderboardevents.TagSwapRequestedPayloadV1
		targetTag          sharedtypes.TagNumber
		mockSetup          func(*leaderboarddb.MockRepository, sharedtypes.GuildID)
		expectedSuccess    bool
		expectedFailureStr string
		expectError        bool
	}{
		{
			name:    "Successful tag swap",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			targetTag: sharedtypes.TagNumber(2),
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
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
			expectedSuccess:    true,
			expectedFailureStr: "",
			expectError:        false,
		},
		{
			name:    "Cannot swap tag with self",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    requestorID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				// Return a leaderboard where the requestor already holds the target tag
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(1),
			expectedSuccess:    false,
			expectedFailureStr: "cannot swap tag with self",
			expectError:        false,
		},
		{
			name:    "No active leaderboard found",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(0),
			expectedSuccess:    false,
			expectedFailureStr: "database error",
			expectError:        true,
		},
		{
			name:    "Database error while fetching leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(nil, errors.New("db error")).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(0),
			expectedSuccess:    false,
			expectedFailureStr: "database error",
			expectError:        true,
		},
		{
			name:    "Requestor does not have a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: nonExistentID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(2),
			expectedSuccess:    false,
			expectedFailureStr: "requesting user not on leaderboard",
			expectError:        false,
		},
		{
			name:    "Target does not have a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    nonExistentID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
					},
				}, nil).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(2),
			expectedSuccess:    false,
			expectedFailureStr: "target tag not currently assigned",
			expectError:        false,
		},
		{
			name:    "Neither user has a tag",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: nonExistentID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
				}, nil).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(2),
			expectedSuccess:    false,
			expectedFailureStr: "requesting user not on leaderboard",
			expectError:        false,
		},
		{
			name:    "Database error while swapping tags",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: leaderboardevents.TagSwapRequestedPayloadV1{
				RequestorID: requestorID,
				TargetID:    targetID,
			},
			mockSetup: func(mdb *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mdb.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{UserID: requestorID, TagNumber: 1},
						{UserID: targetID, TagNumber: 2},
					},
				}, nil).Times(1)
				mdb.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), guildID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).Times(1)
			},
			targetTag:          sharedtypes.TagNumber(2),
			expectedSuccess:    false,
			expectedFailureStr: "database error",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for closure
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := leaderboarddb.NewMockRepository(ctrl)
			s.repo = mockDB
			defer ctrl.Finish()

			tt.mockSetup(mockDB, tt.guildID)

			got, err := s.TagSwapRequested(ctx, tt.guildID, tt.payload.RequestorID, tt.targetTag)

			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: got %v, expected error? %v", err, tt.expectError)
			}

			if tt.expectedSuccess {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !got.IsSuccess() {
					t.Fatalf("expected success result, got failure: %v", got.Failure)
				}
				payload, ok := got.Success.(*leaderboardevents.TagSwapProcessedPayloadV1)
				if !ok {
					t.Fatalf("unexpected success payload type: %T", got.Success)
				}
				if payload.RequestorID != tt.payload.RequestorID || payload.TargetID != tt.payload.TargetID {
					t.Fatalf("unexpected payload values: %+v", payload)
				}
			}

			if tt.expectedFailureStr != "" {
				if !got.IsFailure() {
					t.Fatalf("expected failure result, got success: %v", got.Success)
				}
				payload, ok := got.Failure.(*leaderboardevents.TagSwapFailedPayloadV1)
				if !ok {
					t.Fatalf("unexpected failure payload type: %T", got.Failure)
				}
				if !strings.Contains(payload.Reason, tt.expectedFailureStr) {
					t.Fatalf("expected failure reason to contain %q, got %q", tt.expectedFailureStr, payload.Reason)
				}
			}
		})
	}
}
