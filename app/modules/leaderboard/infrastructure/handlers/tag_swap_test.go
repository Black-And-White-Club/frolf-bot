package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagSwapRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRequestorID := sharedtypes.DiscordID("2468")
	testTargetID := sharedtypes.DiscordID("13579")
	testGuildID := sharedtypes.GuildID("9999")

	testPayload := &leaderboardevents.TagSwapRequestedPayloadV1{
		GuildID:     testGuildID,
		RequestorID: testRequestorID,
		TargetID:    testTargetID,
	}

	// Mock dependencies
	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *leaderboardevents.TagSwapRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully handle TagSwapRequested",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagSwapProcessedPayloadV1{
							GuildID:     testGuildID,
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.TagSwapProcessedV1,
		},
		{
			name: "Service error in TagSwapRequested",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagSwapFailedPayloadV1{
							GuildID:     testGuildID,
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
							Reason:      "cannot swap tag with self",
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.TagSwapFailedV1,
		},
		{
			name: "Unknown result from TagSwapRequested",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
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
			tt.mockSetup()

			h := &LeaderboardHandlers{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
				tracer:             tracer,
				metrics:            metrics,
			}

			ctx := context.Background()
			results, err := h.HandleTagSwapRequested(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleTagSwapRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if !tt.wantErr && tt.wantResultLen > 0 && results[0].Topic != tt.wantTopic {
				t.Errorf("HandleTagSwapRequested() topic = %s, want %s", results[0].Topic, tt.wantTopic)
			}
		})
	}
}
