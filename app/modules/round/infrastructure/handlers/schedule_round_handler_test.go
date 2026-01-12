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

func TestRoundHandlers_HandleDiscordMessageIDUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Test Round")
	testStartTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
	testUserID := sharedtypes.DiscordID("user-123")
	testEventMessageID := "discord-msg-123"

	testPayload := &roundevents.RoundScheduledPayloadV1{
		GuildID: testGuildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:   testRoundID,
			Title:     testTitle,
			StartTime: &testStartTime,
			UserID:    testUserID,
		},
		EventMessageID: testEventMessageID,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundScheduledPayloadV1
		wantErr         bool
		wantResultLen   int
		expectedErrMsg  string
	}{
		{
			name: "Successfully schedule round events",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateSuccessPayloadV1{},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0, // No downstream events for scheduling
		},
		{
			name: "Service returns failure",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundScheduleFailedPayloadV1{
							RoundID: testRoundID,
							Error:   "scheduling failed",
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					testEventMessageID,
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
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					testEventMessageID,
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
			name: "Successfully schedule with description and location",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateSuccessPayloadV1{},
					},
					nil,
				)
			},
			payload: &roundevents.RoundScheduledPayloadV1{
				GuildID: testGuildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       testTitle,
					Description: func() *roundtypes.Description { d := roundtypes.Description("Test Description"); return &d }(),
					Location:    func() *roundtypes.Location { l := roundtypes.Location("Test Location"); return &l }(),
					StartTime:   &testStartTime,
					UserID:      testUserID,
				},
				EventMessageID: testEventMessageID,
			},
			wantErr:       false,
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
			results, err := h.HandleDiscordMessageIDUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleDiscordMessageIDUpdated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
		})
	}
}
