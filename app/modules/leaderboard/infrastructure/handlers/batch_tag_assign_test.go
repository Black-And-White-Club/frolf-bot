package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
)

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testBatchID := uuid.New().String()

	tests := []struct {
		name          string
		setupFake     func(f *FakeService)
		wantErr       bool
		wantResultLen int
	}{
		{
			name: "Successfully assign batch tags",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (leaderboardtypes.LeaderboardData, error) {
					// Service returns domain data (the slice of new assignments)
					return leaderboardtypes.LeaderboardData{
						{UserID: "user-456", TagNumber: 1},
					}, nil
				}
			},
			wantErr: false,
			// mapSuccessResults likely produces 1 'batch_assigned' event + 1 'tag_updated' event per assignment
			wantResultLen: 2,
		},
		{
			name: "Service Infrastructure Error",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (leaderboardtypes.LeaderboardData, error) {
					return nil, fmt.Errorf("connection refused")
				}
			},
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Tag Swap Required Error",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (leaderboardtypes.LeaderboardData, error) {
					return nil, &leaderboardservice.TagSwapNeededError{
						RequestorID: "user-456",
						CurrentTag:  5,
						TargetTag:   1,
					}
				}
			},
			wantErr:       false, // Should return empty results and hand off to saga
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			fakeSaga := NewFakeSagaCoordinator()
			tt.setupFake(fakeSvc)

			h := &LeaderboardHandlers{
				service:         fakeSvc,
				sagaCoordinator: fakeSaga,
			}

			payload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-456", TagNumber: 1}},
			}

			res, err := h.HandleBatchTagAssignmentRequested(context.Background(), payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			if len(res) != tt.wantResultLen {
				t.Errorf("got %d results, want %d", len(res), tt.wantResultLen)
			}

			// Verification for the Saga flow
			if tt.name == "Tag Swap Required Error" {
				if len(fakeSaga.Trace()) == 0 {
					t.Error("Expected saga coordinator to be called but trace is empty")
				}
			}
		})
	}
}
