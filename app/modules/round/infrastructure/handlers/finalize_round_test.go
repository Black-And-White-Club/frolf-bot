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

func scorePointer(s sharedtypes.Score) *sharedtypes.Score {
	return &s
}

func TestRoundHandlers_HandleAllScoresSubmitted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now().UTC())
	testGuildID := sharedtypes.GuildID("guild-123")
	testEventMessageID := "msg-123"

	testParticipants := []roundtypes.Participant{
		{
			UserID:   sharedtypes.DiscordID("user1"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(60)),
		},
		{
			UserID:   sharedtypes.DiscordID("user2"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(65)),
		},
	}

	testPayload := &roundevents.AllScoresSubmittedPayloadV1{
		GuildID:        testGuildID,
		RoundID:        testRoundID,
		EventMessageID: testEventMessageID,
		RoundData: roundtypes.Round{
			ID:             testRoundID,
			Title:          testTitle,
			Location:       &testLocation,
			StartTime:      &testStartTime,
			EventMessageID: testEventMessageID,
			Participants:   testParticipants,
		},
		Participants: testParticipants,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.AllScoresSubmittedPayloadV1
		wantErr         bool
		wantResultLen   int
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle AllScoresSubmitted",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							GuildID: testGuildID,
							RoundID: testRoundID,
							RoundData: roundtypes.Round{
								ID:       testRoundID,
								Title:    testTitle,
								Location: &testLocation,
							},
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2, // Discord + Backend finalization
		},
		{
			name: "Service returns finalization failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "finalization failed",
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1, // Error event
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
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
				mockRoundService.EXPECT().FinalizeRound(
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
			name: "Payload with no GuildID",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				payloadNoGuild := &roundevents.AllScoresSubmittedPayloadV1{
					GuildID:        "", // Empty GuildID
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					RoundData: roundtypes.Round{
						ID:             testRoundID,
						Title:          testTitle,
						Location:       &testLocation,
						StartTime:      &testStartTime,
						EventMessageID: testEventMessageID,
						Participants:   testParticipants,
					},
					Participants: testParticipants,
				}

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*payloadNoGuild,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							GuildID: "",
							RoundID: testRoundID,
						},
					},
					nil,
				)
			},
			payload: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        "",
				RoundID:        testRoundID,
				EventMessageID: testEventMessageID,
				RoundData: roundtypes.Round{
					ID:             testRoundID,
					Title:          testTitle,
					Location:       &testLocation,
					StartTime:      &testStartTime,
					EventMessageID: testEventMessageID,
					Participants:   testParticipants,
				},
				Participants: testParticipants,
			},
			wantErr:       false,
			wantResultLen: 2,
		},
		{
			name: "Payload with multiple participants",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				manyParticipants := []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
					{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
					{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
				}

				payloadMany := &roundevents.AllScoresSubmittedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					RoundData: roundtypes.Round{
						ID:             testRoundID,
						Title:          testTitle,
						EventMessageID: testEventMessageID,
						Participants:   manyParticipants,
					},
					Participants: manyParticipants,
				}

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*payloadMany,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							GuildID: testGuildID,
							RoundID: testRoundID,
						},
					},
					nil,
				)
			},
			payload: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				EventMessageID: testEventMessageID,
				RoundData: roundtypes.Round{
					ID:             testRoundID,
					Title:          testTitle,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
						{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
						{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
					},
				},
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
					{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
					{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
				},
			},
			wantErr:       false,
			wantResultLen: 2,
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
			results, err := h.HandleAllScoresSubmitted(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAllScoresSubmitted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleAllScoresSubmitted() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleAllScoresSubmitted() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			// Verify metadata for Discord message
			if tt.wantResultLen == 2 {
				if results[0].Topic != roundevents.RoundFinalizedDiscordV1 {
					t.Errorf("First result should be Discord finalization, got %v", results[0].Topic)
				}
				if results[0].Metadata["message_id"] != testEventMessageID {
					t.Errorf("Discord message ID metadata not set correctly")
				}
				if results[1].Topic != roundevents.RoundFinalizedV1 {
					t.Errorf("Second result should be backend finalization, got %v", results[1].Topic)
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-456")

	testParticipants := []roundtypes.Participant{
		{
			UserID:   sharedtypes.DiscordID("user1"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(60)),
		},
		{
			UserID:   sharedtypes.DiscordID("user2"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(65)),
		},
	}

	testPayload := &roundevents.RoundFinalizedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		RoundData: roundtypes.Round{
			ID:           testRoundID,
			Participants: testParticipants,
		},
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundFinalizedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundFinalized with score processing",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ProcessRoundScoresRequestPayloadV1{
							RoundID: testRoundID,
							Scores: []roundevents.ParticipantScoreV1{
								{UserID: sharedtypes.DiscordID("user1"), Score: sharedtypes.Score(60)},
								{UserID: sharedtypes.DiscordID("user2"), Score: sharedtypes.Score(65)},
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ProcessRoundScoresRequestedV1,
		},
		{
			name: "Service returns finalization error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "no participants with scores",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundFinalizationErrorV1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service unavailable"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service unavailable",
		},
		{
			name: "Service returns empty result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().NotifyScoreModule(
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
				mockRoundService.EXPECT().NotifyScoreModule(
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
		{
			name: "Empty participants list",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				payloadNoParticipants := &roundevents.RoundFinalizedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					RoundData: roundtypes.Round{
						ID:           testRoundID,
						Participants: []roundtypes.Participant{},
					},
				}

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*payloadNoParticipants,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "no participants",
						},
					},
					nil,
				)
			},
			payload: &roundevents.RoundFinalizedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: []roundtypes.Participant{},
				},
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundFinalizationErrorV1,
		},
		{
			name: "Multiple scores for processing",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				manyParticipants := []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
					{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
				}

				payloadMany := &roundevents.RoundFinalizedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					RoundData: roundtypes.Round{
						ID:           testRoundID,
						Participants: manyParticipants,
					},
				}

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*payloadMany,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ProcessRoundScoresRequestPayloadV1{
							RoundID: testRoundID,
							Scores: []roundevents.ParticipantScoreV1{
								{UserID: sharedtypes.DiscordID("user1"), Score: sharedtypes.Score(60)},
								{UserID: sharedtypes.DiscordID("user2"), Score: sharedtypes.Score(65)},
								{UserID: sharedtypes.DiscordID("user3"), Score: sharedtypes.Score(55)},
							},
						},
					},
					nil,
				)
			},
			payload: &roundevents.RoundFinalizedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID: testRoundID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
						{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
					},
				},
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ProcessRoundScoresRequestedV1,
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
			results, err := h.HandleRoundFinalized(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundFinalized() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundFinalized() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundFinalized() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
