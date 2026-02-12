package leaderboardhandlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{
							"12345678901234567": 1,
							"12345678901234568": 2,
						},
						PointsSkipped: true,
					}, nil
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
			name: "Command handler returns error",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return nil, fmt.Errorf("command failed")
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service Infrastructure Error",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return nil, fmt.Errorf("database down")
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Invalid tag format in payload - skips bad entries",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					if len(cmd.Participants) != 1 {
						return nil, fmt.Errorf("expected 1 participant, got %d", len(cmd.Participants))
					}
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{
							"12345": 1,
						},
						PointsSkipped: true,
					}, nil
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
		{
			name: "Uses explicit participants when present",
			setupFake: func(f *FakeService, s *FakeSagaCoordinator, u *FakeUserService) {
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					if len(cmd.Participants) != 2 {
						return nil, fmt.Errorf("expected 2 participants, got %d", len(cmd.Participants))
					}
					if cmd.Participants[0].MemberID != "98765432100000001" || cmd.Participants[0].FinishRank != 1 {
						return nil, fmt.Errorf("unexpected first participant: %+v", cmd.Participants[0])
					}
					if cmd.Participants[1].MemberID != "98765432100000002" || cmd.Participants[1].FinishRank != 2 {
						return nil, fmt.Errorf("unexpected second participant: %+v", cmd.Participants[1])
					}
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{
							"98765432100000001": 1,
							"98765432100000002": 2,
						},
						PointsSkipped: true,
					}, nil
				}
			},
			payload: &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Participants: []leaderboardevents.RoundParticipantInputV1{
					{MemberID: "98765432100000001", FinishRank: 1},
					{MemberID: "98765432100000002", FinishRank: 2},
				},
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
