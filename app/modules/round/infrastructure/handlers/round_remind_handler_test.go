package roundhandlers

import (
	"context"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testReminderType := "24-hour"
	testUserIDs := []sharedtypes.DiscordID{"user1", "user2", "user3"}

	testPayload := &roundevents.DiscordReminderPayloadV1{
		RoundID:      testRoundID,
		GuildID:      testGuildID,
		ReminderType: testReminderType,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.DiscordReminderPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle round reminder with participants",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.DiscordReminderPayloadV1{
							RoundID:      testRoundID,
							GuildID:      testGuildID,
							ReminderType: testReminderType,
							UserIDs:      testUserIDs,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderSentV1,
		},
		{
			name: "Successfully handle round reminder with no participants",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.DiscordReminderPayloadV1{
							RoundID:      testRoundID,
							GuildID:      testGuildID,
							ReminderType: testReminderType,
							UserIDs:      []sharedtypes.DiscordID{}, // No participants
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0, // Returns empty results when no participants
		},
		{
			name: "Service returns failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundReminderFailedPayloadV1{
							RoundID: testRoundID,
							Error:   "round not found",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderFailedV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("database connection failed"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection failed",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service returns unexpected payload type",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{}, // Wrong type
					},
					nil,
				)
			},
			payload: testPayload,
			wantErr: true,
		},
		{
			name: "Successfully handle with single participant",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.DiscordReminderPayloadV1{
							RoundID:      testRoundID,
							GuildID:      testGuildID,
							ReminderType: testReminderType,
							UserIDs:      []sharedtypes.DiscordID{"user1"},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderSentV1,
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
			results, err := h.HandleRoundReminder(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundReminder() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundReminder() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundReminder() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
