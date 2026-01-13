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

func TestRoundHandlers_HandleScoreUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(42)

	testPayload := &roundevents.ScoreUpdateRequestPayloadV1{
		GuildID:   sharedtypes.GuildID("test-guild"),
		RoundID:   testRoundID,
		UserID:    testParticipant,
		Score:     &testScore,
		ChannelID: "test-channel",
		MessageID: "test-message",
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ScoreUpdateRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScoreUpdateRequest with validation success",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScoreUpdateValidatedPayloadV1{
							ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
								GuildID:   sharedtypes.GuildID("test-guild"),
								RoundID:   testRoundID,
								UserID:    testParticipant,
								Score:     &testScore,
								ChannelID: "test-channel",
								MessageID: "test-message",
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateValidatedV1,
		},
		{
			name: "Service returns validation failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
							Error:   "validation failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateErrorV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "internal service error",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					*testPayload,
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
			name: "Score is nil",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				payloadWithoutScore := &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     nil,
					ChannelID: "test-channel",
					MessageID: "test-message",
				}

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					*payloadWithoutScore,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
							Error:   "score cannot be nil",
						},
					},
					nil,
				)
			},
			payload: &roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   testRoundID,
				UserID:    testParticipant,
				Score:     nil,
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateErrorV1,
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
			results, err := h.HandleScoreUpdateRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScoreUpdateRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScoreUpdateRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(50)

	testPayload := &roundevents.ScoreUpdateValidatedPayloadV1{
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:   sharedtypes.GuildID("test-guild"),
			RoundID:   testRoundID,
			UserID:    testParticipant,
			Score:     &testScore,
			ChannelID: "test-channel",
			MessageID: "test-message",
		},
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ScoreUpdateValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScoreUpdateValidated",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantScoreUpdatedPayloadV1{
							RoundID: testRoundID,
							UserID:  testParticipant,
							Score:   testScore,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundParticipantScoreUpdatedV1,
		},
		{
			name: "Service returns failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
							Error:   "database error",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateErrorV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("connection failed"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "connection failed",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					*testPayload,
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
				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{}, // Wrong type
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantResultTopic: roundevents.RoundParticipantScoreUpdatedV1,
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
			results, err := h.HandleScoreUpdateValidated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateValidated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScoreUpdateValidated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScoreUpdateValidated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(45)
	testEventMessageID := "msg-12345"

	testPayload := &roundevents.ParticipantScoreUpdatedPayloadV1{
		RoundID:        testRoundID,
		UserID:         testParticipant,
		Score:          testScore,
		ChannelID:      "test-channel",
		EventMessageID: testEventMessageID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ParticipantScoreUpdatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "All scores submitted - success path",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.AllScoresSubmittedPayloadV1{
							RoundID: testRoundID,
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
			name: "Not all scores submitted yet - partial path",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScoresPartiallySubmittedPayloadV1{
							RoundID:              testRoundID,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoresPartiallySubmittedV1,
		},
		{
			name: "Service returns failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationFailedPayloadV1{
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
			wantResultTopic: roundevents.RoundFinalizationFailedV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("database connection lost"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection lost",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
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
				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					*testPayload,
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
			results, err := h.HandleParticipantScoreUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleParticipantScoreUpdated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleParticipantScoreUpdated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
