package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testPayload := &leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: testGuildID,
	}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *leaderboardevents.GetLeaderboardRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully get leaderboard",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetLeaderboard(gomock.Any(), testGuildID).Return(
					results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{
						GuildID:     testGuildID,
						Leaderboard: []leaderboardtypes.LeaderboardEntry{},
					}),
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.GetLeaderboardResponseV1,
		},
		{
			name: "Service error in GetLeaderboard",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetLeaderboard(gomock.Any(), testGuildID).Return(
					results.OperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service failure - no active leaderboard",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetLeaderboard(gomock.Any(), testGuildID).Return(
					results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{
						GuildID:     testGuildID,
						Leaderboard: []leaderboardtypes.LeaderboardEntry{},
					}),
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.GetLeaderboardResponseV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				service: mockLeaderboardService,
			}

			ctx := context.Background()
			results, err := h.HandleGetLeaderboardRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleGetLeaderboardRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if !tt.wantErr && tt.wantResultLen > 0 && results[0].Topic != tt.wantTopic {
				t.Errorf("HandleGetLeaderboardRequest() topic = %s, want %s", results[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testTagNumber := sharedtypes.TagNumber(5)
	testPayload := &sharedevents.DiscordTagLookupRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		UserID:        testUserID,
	}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *sharedevents.DiscordTagLookupRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully lookup tag - found",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testPayload.GuildID, testPayload.UserID).Return(
					testTagNumber,
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     sharedevents.LeaderboardTagLookupSucceededV1,
		},
		{
			name: "Successfully lookup tag - not found",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testPayload.GuildID, testPayload.UserID).Return(
					sharedtypes.TagNumber(0),
					fmt.Errorf("not found"),
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     sharedevents.LeaderboardTagLookupNotFoundV1,
		},
		{
			name: "Service error",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testPayload.GuildID, testPayload.UserID).Return(
					sharedtypes.TagNumber(0),
					fmt.Errorf("service error"),
				)
			},
			payload: testPayload,
			// Handler treats service errors as "not found" for this lookup and returns a not-found event
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     sharedevents.LeaderboardTagLookupNotFoundV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				service: mockLeaderboardService,
			}

			ctx := context.Background()
			results, err := h.HandleGetTagByUserIDRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByUserIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleGetTagByUserIDRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if !tt.wantErr && tt.wantResultLen > 0 && results[0].Topic != tt.wantTopic {
				t.Errorf("HandleGetTagByUserIDRequest() topic = %s, want %s", results[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleRoundGetTagRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("user-456")
	testTagNumber := sharedtypes.TagNumber(3)
	joinedLateFalse := false
	testPayload := &sharedevents.RoundTagLookupRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		RoundID:       testRoundID,
		UserID:        testUserID,
		Response:      "yes",
		JoinedLate:    &joinedLateFalse,
	}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *sharedevents.RoundTagLookupRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully lookup round tag - found",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(gomock.Any(), testPayload.GuildID, *testPayload).Return(
					results.SuccessResult(&sharedevents.RoundTagLookupResultPayloadV1{
						ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testPayload.GuildID},
						UserID:        testUserID,
						TagNumber:     &testTagNumber,
						Found:         true,
					}),
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     sharedevents.RoundTagLookupFoundV1,
		},
		{
			name: "Successfully lookup round tag - not found",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(gomock.Any(), testPayload.GuildID, *testPayload).Return(
					results.FailureResult(&sharedevents.RoundTagLookupFailedPayloadV1{
						ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testPayload.GuildID},
						UserID:        testUserID,
						RoundID:       testPayload.RoundID,
						Reason:        "not found",
					}),
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     sharedevents.RoundTagLookupFailedV1,
		},
		{
			name: "Service error",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(gomock.Any(), testPayload.GuildID, *testPayload).Return(
					results.OperationResult{},
					fmt.Errorf("service error"),
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				service: mockLeaderboardService,
			}

			ctx := context.Background()
			results, err := h.HandleRoundGetTagRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundGetTagRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundGetTagRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if !tt.wantErr && tt.wantResultLen > 0 && results[0].Topic != tt.wantTopic {
				t.Errorf("HandleRoundGetTagRequest() topic = %s, want %s", results[0].Topic, tt.wantTopic)
			}
		})
	}
}
