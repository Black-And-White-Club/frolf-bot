package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testBatchID := uuid.New().String()
	testPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: testGuildID},
		RequestingUserID: sharedtypes.DiscordID("user-123"),
		BatchID:          testBatchID,
		Assignments: []sharedevents.TagAssignmentInfoV1{
			{
				UserID:    sharedtypes.DiscordID("user-456"),
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
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.SuccessResult(&leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
						GuildID: testGuildID,
						Assignments: []leaderboardevents.TagAssignmentInfoV1{
							{UserID: sharedtypes.DiscordID("user-456"), TagNumber: sharedtypes.TagNumber(1)},
						},
					}),
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
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.FailureResult(&leaderboardevents.LeaderboardBatchTagAssignmentFailedPayloadV1{GuildID: testGuildID, Reason: "service error"}),
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
				service:         mockLeaderboardService,
				sagaCoordinator: nil,
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
