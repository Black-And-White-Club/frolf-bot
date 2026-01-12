package roundhandlers

import (
	"context"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantJoinRequest_Basic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("user-123")

	testPayload := &roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:  testRoundID,
		GuildID:  testGuildID,
		UserID:   testUserID,
		Response: roundtypes.ResponseAccept,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService)
		payload        *roundevents.ParticipantJoinRequestPayloadV1
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully validate participant join request",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantJoinValidationRequestPayloadV1{
							RoundID:  testRoundID,
							GuildID:  testGuildID,
							UserID:   testUserID,
							Response: roundtypes.ResponseAccept,
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
				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
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

			_, err := h.HandleParticipantJoinRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantJoinRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
		})
	}
}
