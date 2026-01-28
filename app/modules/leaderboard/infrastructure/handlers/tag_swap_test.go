package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

func TestLeaderboardHandlers_HandleTagSwapRequested(t *testing.T) {
	testRequestorID := sharedtypes.DiscordID("2468")
	testTargetID := sharedtypes.DiscordID("13579")
	testGuildID := sharedtypes.GuildID("9999")

	testPayload := &leaderboardevents.TagSwapRequestedPayloadV1{
		GuildID:     testGuildID,
		RequestorID: testRequestorID,
		TargetID:    testTargetID,
	}

	tests := []struct {
		name          string
		setupFake     func(f *FakeService, s *FakeSagaCoordinator)
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully handle TagSwapRequested - Immediate Success",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator) {
				// Target currently holds tag 2
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					if u == testTargetID {
						return results.SuccessResult[sharedtypes.TagNumber, error](2), nil
					}
					return results.SuccessResult[sharedtypes.TagNumber, error](1), nil // Requestor holds 1
				}
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, g sharedtypes.GuildID, r []sharedtypes.TagAssignmentRequest, u sharedtypes.RoundID, s sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{
						{UserID: testRequestorID, TagNumber: 2},
						{UserID: testTargetID, TagNumber: 1},
					}), nil
				}
			},
			wantErr:       false,
			wantResultLen: 2, // mapSuccessResults likely produces batch_assigned + tag_updated + guild_scoped
			wantTopic:     leaderboardevents.LeaderboardBatchTagAssignedV1,
		},
		{
			name: "Tag Swap Needed - Triggers Saga",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.SuccessResult[sharedtypes.TagNumber, error](2), nil
				}
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, g sharedtypes.GuildID, r []sharedtypes.TagAssignmentRequest, u sharedtypes.RoundID, src sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					err := error(&leaderboardservice.TagSwapNeededError{
						RequestorID: testRequestorID,
						TargetTag:   2,
						CurrentTag:  1,
					})
					return results.FailureResult[leaderboardtypes.LeaderboardData, error](err), nil
				}
			},
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.TagSwapProcessedV1,
		},
		{
			name: "Target User Has No Tag",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.FailureResult[sharedtypes.TagNumber, error](fmt.Errorf("not found")), nil
				}
			},
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.TagSwapFailedV1,
		},
		{
			name: "Service Infrastructure Error",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.SuccessResult[sharedtypes.TagNumber, error](2), nil
				}
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, g sharedtypes.GuildID, r []sharedtypes.TagAssignmentRequest, u sharedtypes.RoundID, src sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, fmt.Errorf("database timeout")
				}
			},
			wantErr:       true,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			fakeSaga := NewFakeSagaCoordinator()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc, fakeSaga)
			}

			h := &LeaderboardHandlers{
				service:         fakeSvc,
				sagaCoordinator: fakeSaga,
			}

			res, err := h.HandleTagSwapRequested(context.Background(), testPayload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleTagSwapRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(res) != tt.wantResultLen {
				t.Errorf("Result length = %d, want %d", len(res), tt.wantResultLen)
			}

			if !tt.wantErr && len(res) > 0 && res[0].Topic != tt.wantTopic {
				t.Errorf("First result topic = %s, want %s", res[0].Topic, tt.wantTopic)
			}

			// Specific verification for the Saga flow
			if tt.wantTopic == leaderboardevents.TagSwapProcessedV1 {
				if len(fakeSaga.CapturedIntents) == 0 {
					t.Error("Expected saga coordinator to capture intent, but it was empty")
				}
			}
		})
	}
}
