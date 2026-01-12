package roundhandlers

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func TestRoundHandlers_HandleGetRoundRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("Test Description")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
	testUserID := sharedtypes.DiscordID("user-123")

	testPayload := &roundevents.GetRoundRequestPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.GetRoundRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully retrieve round",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundtypes.Round{
							ID:          testRoundID,
							Title:       testTitle,
							Description: &testDescription,
							Location:    &testLocation,
							StartTime:   &testStartTime,
							CreatedBy:   testUserID,
							GuildID:     testGuildID,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
		},
		{
			name: "Service returns failure - round not found",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundRetrievalFailedPayloadV1{
							GuildID: testGuildID,
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
			wantResultTopic: roundevents.RoundRetrievalFailedV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("database connection error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection error",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{},
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
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{}, // Wrong type
					},
					nil,
				)
			},
			payload: testPayload,
			wantErr: true,
		},
		{
			name: "Successfully retrieve minimal round data",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundtypes.Round{
							ID:      testRoundID,
							Title:   testTitle,
							GuildID: testGuildID,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
		},
		{
			name: "Successfully retrieve round with participants",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				testScore := sharedtypes.Score(65)
				mockRoundService.EXPECT().GetRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundtypes.Round{
							ID:          testRoundID,
							Title:       testTitle,
							Description: &testDescription,
							Location:    &testLocation,
							StartTime:   &testStartTime,
							CreatedBy:   testUserID,
							GuildID:     testGuildID,
							Participants: []roundtypes.Participant{
								{
									UserID:   sharedtypes.DiscordID("user1"),
									Response: roundtypes.ResponseAccept,
									Score:    &testScore,
								},
								{
									UserID:   sharedtypes.DiscordID("user2"),
									Response: roundtypes.ResponseDecline,
								},
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
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

			ctx := context.Background()
			results, err := h.HandleGetRoundRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetRoundRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetRoundRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleGetRoundRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleGetRoundRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
