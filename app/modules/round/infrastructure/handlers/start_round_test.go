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

func TestRoundHandlers_HandleRoundStartRequested_Basic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")

	testPayload := &roundevents.RoundStartRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService)
		payload        *roundevents.RoundStartRequestedPayloadV1
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundStarted",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundStart(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					results.OperationResult{
						Success: &roundevents.DiscordRoundStartPayloadV1{
							RoundID:      testRoundID,
							GuildID:      testGuildID,
							Title:        "Test Round",
							Participants: []roundevents.RoundParticipantV1{{UserID: "user1"}, {UserID: "user2"}},
						},
					},
					nil,
				)
			},
			payload: testPayload,
			wantErr: false,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundStart(
					gomock.Any(),
					testGuildID,
					testRoundID,
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
				mockRoundService.EXPECT().ProcessRoundStart(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload: testPayload,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				service: mockRoundService,
				logger:  logger,
			}

			_, err := h.HandleRoundStartRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStartRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundStartRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
		})
	}
}
