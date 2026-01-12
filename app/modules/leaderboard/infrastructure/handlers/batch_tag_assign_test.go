package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testBatchID := uuid.New().String()
	testPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		RequestingUserID: sharedtypes.DiscordID("user-123"),
		BatchID: testBatchID,
		Assignments: []sharedevents.TagAssignmentInfoV1{
			{
				UserID: sharedtypes.DiscordID("user-456"),
				TagNumber: sharedtypes.TagNumber(1),
			},
		},
	}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *sharedevents.BatchTagAssignmentRequestedPayloadV1
		wantErr       bool
		wantResultLen int
	}{
		{
			name: "Successfully assign batch tags",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					testGuildID,
					testPayload,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
							GuildID: testGuildID,
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2, // success + tag update
		},
		{
			name: "Service error",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					testGuildID,
					testPayload,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("service error"),
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
			results, err := h.HandleBatchTagAssignmentRequested(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBatchTagAssignmentRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleBatchTagAssignmentRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}
		})
	}
}
