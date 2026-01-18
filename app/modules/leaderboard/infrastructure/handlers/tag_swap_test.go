package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
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
				// Target currently holds a tag 2
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(2), nil)
				// Requestor currently holds tag 1
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(1), nil)
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
							{UserID: testRequestorID, TagNumber: 2},
							{UserID: testTargetID, TagNumber: 1},
						},
					}),
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2,
			wantTopic:     leaderboardevents.LeaderboardBatchTagAssignedV1,
		},
		{
			name: "Service error in TagSwapRequested",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(2), nil)
				// Requestor tag lookup (handler collects but ignores error) - provide a value
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(1), nil)
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(results.OperationResult{}, fmt.Errorf("internal service error"))
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				// Target has no tag
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(0), fmt.Errorf("not found"))
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.TagSwapFailedV1,
		},
		{
			name: "Unknown result from TagSwapRequested",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(2), nil)
				mockLeaderboardService.EXPECT().GetTagByUserID(gomock.Any(), testGuildID, gomock.Any()).Return(sharedtypes.TagNumber(0), fmt.Errorf("not found"))
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					testGuildID,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(results.OperationResult{}, nil)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2,
			wantTopic:     leaderboardevents.LeaderboardBatchTagAssignedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				service: mockLeaderboardService,
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
