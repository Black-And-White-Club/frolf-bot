package roundhandlers

import (
	"context"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundStarted_Basic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")

	testPayload := &roundevents.RoundStartedPayloadV1{
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService)
		payload        *roundevents.RoundStartedPayloadV1
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundStarted",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ProcessRoundStart(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.DiscordRoundStartPayloadV1{
							RoundID:     testRoundID,
							GuildID:     testGuildID,
							Title:       "Test Round",
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
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
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
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			payload: testPayload,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			_, err := h.HandleRoundStarted(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStarted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundStarted() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
		})
	}
}
