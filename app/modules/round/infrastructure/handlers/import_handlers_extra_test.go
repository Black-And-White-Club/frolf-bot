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

func TestRoundHandlers_HandleImportCompleted(t *testing.T) {
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-1"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.ImportCompletedPayloadV1{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{{UserID: sharedtypes.DiscordID("u1"), Score: 5}},
	}

	scorePointer := func(s sharedtypes.Score) *sharedtypes.Score {
		return &s
	}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ImportCompletedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "FanOut messages successfully",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				finalParticipants := []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("u1"), Score: scorePointer(sharedtypes.Score(5))},
				}
				mockRoundService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ImportScoresAppliedPayloadV1{
							GuildID:        guildID,
							RoundID:        roundID,
							ImportID:       importID,
							Participants:   finalParticipants,
							EventMessageID: "evt-1",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundAllScoresSubmittedV1,
		},
		{
			name: "Service failure produces ImportFailed",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{
							GuildID:  guildID,
							RoundID:  roundID,
							ImportID: importID,
							Error:    "all failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
		{
			name: "No scores returns nil",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				// Service should NOT be called when there are no scores
				mockRoundService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Times(0)
			},
			payload: &roundevents.ImportCompletedPayloadV1{
				ImportID: importID,
				GuildID:  guildID,
				RoundID:  roundID,
				Scores:   []sharedtypes.ScoreInfo{},
			},
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Unexpected success type returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
					roundservice.RoundOperationResult{Success: "not-the-right-type"},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       true,
			expectedErrMsg: "unexpected success payload type",
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			results, err := h.HandleImportCompleted(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleImportCompleted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleImportCompleted() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleImportCompleted() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleImportCompleted() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}

			// Verify result payload content for successful cases
			if tt.wantResultLen > 0 && tt.wantResultTopic == roundevents.RoundAllScoresSubmittedV1 {
				resultPayload, ok := results[0].Payload.(*roundevents.AllScoresSubmittedPayloadV1)
				if !ok {
					t.Errorf("HandleImportCompleted() payload type mismatch")
				}
				if resultPayload.GuildID != guildID || resultPayload.RoundID != roundID {
					t.Errorf("HandleImportCompleted() payload data mismatch")
				}
			}
		})
	}
}
