package roundhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleScheduledRoundTagUpdate(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	tn1 := sharedtypes.TagNumber(1)
	tn2 := sharedtypes.TagNumber(13)

	testPayload := &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
		GuildID: testGuildID,
		ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
			sharedtypes.DiscordID("user1"): tn1,
			sharedtypes.DiscordID("user2"): tn2,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *leaderboardevents.TagUpdateForScheduledRoundsPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle scheduled round tag update with changes",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.TagsUpdatedForScheduledRoundsPayloadV1{
							UpdatedRounds: []roundevents.RoundUpdateInfoV1{
								{
									RoundID:             testRoundID,
									Title:               "Test Round",
									EventMessageID:      "msg-123",
									UpdatedParticipants: []roundtypes.Participant{},
									ParticipantsChanged: 2,
								},
							},
							Summary: roundevents.UpdateSummaryV1{
								TotalRoundsProcessed: 1,
								RoundsUpdated:        1,
								ParticipantsUpdated:  2,
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.TagsUpdatedForScheduledRoundsV1,
		},
		{
			name: "Service returns failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							GuildID: testGuildID,
							Error:   "tag update failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Payload with no changed tags returns nil",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				// Service is called by handler; return empty result
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload: &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
				GuildID:     testGuildID,
				ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{},
			},
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Success with no affected rounds returns nil",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.TagsUpdatedForScheduledRoundsPayloadV1{
							UpdatedRounds: []roundevents.RoundUpdateInfoV1{}, // No rounds affected
							Summary: roundevents.UpdateSummaryV1{
								TotalRoundsProcessed: 0,
								RoundsUpdated:        0,
								ParticipantsUpdated:  0,
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.TagsUpdatedForScheduledRoundsV1,
		},
		{
			name: "Service returns unexpected payload type",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{}, // Wrong type
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.TagsUpdatedForScheduledRoundsV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				service: mockRoundService,
				logger:  logger,
			}

			ctx := context.Background()
			results, err := h.HandleScheduledRoundTagUpdate(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScheduledRoundTagUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScheduledRoundTagUpdate() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScheduledRoundTagUpdate() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScheduledRoundTagUpdate() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
