package leaderboardhandlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
)

func TestLeaderboardHandlers_HandleLeaderboardUpdateRequested(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("test-guild")
	testClubUUID := uuid.New()
	testSortedParticipantTags := []string{
		"1:12345678901234567", // 1st place
		"2:12345678901234568", // 2nd place
	}

	testPayload := &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
		GuildID:               testGuildID,
		RoundID:               testRoundID,
		SortedParticipantTags: testSortedParticipantTags,
		Source:                "round",
		UpdateID:              testRoundID.String(),
	}

	tests := []struct {
		name          string
		setupFake     func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService)
		payload       *leaderboardevents.LeaderboardUpdateRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopics    []string
		verifySaga    func(t *testing.T, s *FakeSagaCoordinator)
	}{
		{
			name: "Successfully handle LeaderboardUpdateRequested",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{
						{UserID: "12345678901234567", TagNumber: 1},
						{UserID: "12345678901234568", TagNumber: 2},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 4,
			wantTopics: []string{
				leaderboardevents.LeaderboardUpdatedV1,                                              // 0: Global
				sharedevents.SyncRoundsTagRequestV1,                                                 // 1: Sync
				fmt.Sprintf("%s.%s", leaderboardevents.LeaderboardUpdatedV1, "test-guild"),          // 2: Scoped
				fmt.Sprintf("%s.%s", leaderboardevents.LeaderboardUpdatedV1, testClubUUID.String()), // 3: Scoped Club
			},
		},
		{
			name: "Tag Swap Required - Triggers Saga",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					err := error(&leaderboardservice.TagSwapNeededError{
						RequestorID: "12345678901234567",
						CurrentTag:  5,
						TargetTag:   1,
					})
					return results.FailureResult[leaderboardtypes.LeaderboardData, error](err), nil
				}
			},
			payload:       testPayload,
			wantErr:       false, // Handler returns empty results, not an error, when starting a saga
			wantResultLen: 0,
			verifySaga: func(t *testing.T, s *FakeSagaCoordinator) {
				if len(s.CapturedIntents) != 1 {
					t.Errorf("Expected 1 saga intent, got %d", len(s.CapturedIntents))
				}
			},
		},
		{
			name: "Service Infrastructure Error",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, fmt.Errorf("database down")
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Invalid tag format in payload - skips bad entries",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					// Verify only the valid tag was passed to the service
					if len(requests) != 1 {
						return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, fmt.Errorf("expected 1 request, got %d", len(requests))
					}
					return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{{UserID: "12345", TagNumber: 1}}), nil
				}
			},
			payload: &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
				GuildID:               testGuildID,
				RoundID:               testRoundID,
				SortedParticipantTags: []string{"invalid_format", "1:12345"},
			},
			wantErr:       false,
			wantResultLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			fakeSaga := NewFakeSagaCoordinator()
			fakeUserService := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc, fakeSaga, fakeUserService)
			}

			h := &LeaderboardHandlers{
				service:         fakeSvc,
				userService:     fakeUserService,
				sagaCoordinator: fakeSaga,
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			res, err := h.HandleLeaderboardUpdateRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(res) != tt.wantResultLen {
				t.Errorf("Result length = %d, want %d", len(res), tt.wantResultLen)
			}

			if !tt.wantErr && len(tt.wantTopics) > 0 {
				for i, topic := range tt.wantTopics {
					if i < len(res) && res[i].Topic != topic {
						t.Errorf("Result[%d] topic = %s, want %s", i, res[i].Topic, topic)
					}
				}
			}

			if tt.verifySaga != nil {
				tt.verifySaga(t, fakeSaga)
			}
		})
	}
}
