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

func TestRoundHandlers_HandleRoundDeleteRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              testGuildID,
		RoundID:              testRoundID,
		RequestingUserUserID: testUserID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundDeleteRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundDeleteRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeleteValidatedPayloadV1{
							GuildID:                   testGuildID,
							RoundDeleteRequestPayload: *testPayload,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteValidatedV1,
		},
		{
			name: "Service failure returns delete error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							GuildID: testGuildID,
							RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
								GuildID:              testGuildID,
								RoundID:              testRoundID,
								RequestingUserUserID: testUserID,
							},
							Error: "unauthorized: only the round creator can delete the round",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteErrorV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
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
		{
			name: "Unknown result returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
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
			results, err := h.HandleRoundDeleteRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteValidatedPayloadV1{
		GuildID: testGuildID,
		RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayloadV1{
			GuildID:              testGuildID,
			RoundID:              testRoundID,
			RequestingUserUserID: testUserID,
		},
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		payload         *roundevents.RoundDeleteValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
	}{
		{
			name:            "Successfully transform to RoundDeleteAuthorized",
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteAuthorizedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			ctx := context.Background()
			results, err := h.HandleRoundDeleteValidated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteValidated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteValidated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			if tt.wantResultLen > 0 {
				resultPayload, ok := results[0].Payload.(*roundevents.RoundDeleteAuthorizedPayloadV1)
				if !ok {
					t.Errorf("HandleRoundDeleteValidated() payload type mismatch")
				}
				if resultPayload.GuildID != testGuildID || resultPayload.RoundID != testRoundID {
					t.Errorf("HandleRoundDeleteValidated() payload data mismatch")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteAuthorized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")

	testPayload := &roundevents.RoundDeleteAuthorizedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name             string
		mockSetup        func(*roundmocks.MockService)
		payload          *roundevents.RoundDeleteAuthorizedPayloadV1
		ctx              context.Context
		wantErr          bool
		wantResultLen    int
		wantResultTopic  string
		expectedErrMsg   string
		checkMetadata    bool
		expectedMetadata map[string]string
	}{
		{
			name: "Successfully delete round",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayloadV1{
							GuildID: testGuildID,
							RoundID: testRoundID,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			ctx:             context.Background(),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeletedV1,
		},
		{
			name: "Successfully delete round with discord message ID in metadata",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayloadV1{
							GuildID: testGuildID,
							RoundID: testRoundID,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeletedV1,
			checkMetadata:   true,
			expectedMetadata: map[string]string{
				"discord_message_id": "msg-123",
			},
		},
		{
			name: "Service failure returns delete error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							GuildID: testGuildID,
							RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
								GuildID: testGuildID,
								RoundID: testRoundID,
							},
							Error: "round not found",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			ctx:             context.Background(),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteErrorV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:        testPayload,
			ctx:            context.Background(),
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			ctx:           context.Background(),
			wantErr:       true,
			wantResultLen: 0,
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

			results, err := h.HandleRoundDeleteAuthorized(tt.ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteAuthorized() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteAuthorized() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			if tt.checkMetadata && tt.wantResultLen > 0 {
				for key, expectedValue := range tt.expectedMetadata {
					actualValue, exists := results[0].Metadata[key]
					if !exists {
						t.Errorf("HandleRoundDeleteAuthorized() metadata key %s missing", key)
					}
					if actualValue != expectedValue {
						t.Errorf("HandleRoundDeleteAuthorized() metadata[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			}
		})
	}
}
