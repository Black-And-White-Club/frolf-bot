package leaderboardhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
)

func TestLeaderboardHandlers_HandlePointHistoryRequested(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	memberID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name         string
		payload      *PointHistoryRequestedPayloadV1
		setupFake    func(*FakeService)
		wantTopic    string
		wantErr      bool
	}{
		{
			name:    "success returns history",
			payload: &PointHistoryRequestedPayloadV1{GuildID: guildID, MemberID: memberID, Limit: 10},
			setupFake: func(f *FakeService) {
				f.GetPointHistoryForMemberFunc = func(ctx context.Context, g sharedtypes.GuildID, m sharedtypes.DiscordID, limit int) (results.OperationResult[[]leaderboardservice.PointHistoryEntry, error], error) {
					return results.SuccessResult[[]leaderboardservice.PointHistoryEntry, error]([]leaderboardservice.PointHistoryEntry{
						{MemberID: memberID, Points: 100, Reason: "Round Matchups", Tier: "Gold", Opponents: 3},
					}), nil
				}
			},
			wantTopic: LeaderboardPointHistoryResponseV1,
		},
		{
			name:    "service error returns failure",
			payload: &PointHistoryRequestedPayloadV1{GuildID: guildID, MemberID: memberID},
			setupFake: func(f *FakeService) {
				f.GetPointHistoryForMemberFunc = func(ctx context.Context, g sharedtypes.GuildID, m sharedtypes.DiscordID, limit int) (results.OperationResult[[]leaderboardservice.PointHistoryEntry, error], error) {
					return results.OperationResult[[]leaderboardservice.PointHistoryEntry, error]{}, errors.New("db error")
				}
			},
			wantTopic: LeaderboardPointHistoryFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{
				service: fakeSvc,
				logger:  slog.Default(),
			}

			got, err := h.HandlePointHistoryRequested(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) > 0 && got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleManualPointAdjustment(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	memberID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name      string
		payload   *ManualPointAdjustmentPayloadV1
		setupFake func(*FakeService)
		wantTopic string
	}{
		{
			name: "success adjustment",
			payload: &ManualPointAdjustmentPayloadV1{
				GuildID: guildID, MemberID: memberID, PointsDelta: 50, Reason: "scorecard error", AdminID: "admin-1",
			},
			setupFake: func(f *FakeService) {
				f.AdjustPointsFunc = func(ctx context.Context, g sharedtypes.GuildID, m sharedtypes.DiscordID, delta int, reason string) (results.OperationResult[bool, error], error) {
					return results.SuccessResult[bool, error](true), nil
				}
			},
			wantTopic: LeaderboardManualPointAdjustmentSuccessV1,
		},
		{
			name: "service error",
			payload: &ManualPointAdjustmentPayloadV1{
				GuildID: guildID, MemberID: memberID, PointsDelta: 50, Reason: "test",
			},
			setupFake: func(f *FakeService) {
				f.AdjustPointsFunc = func(ctx context.Context, g sharedtypes.GuildID, m sharedtypes.DiscordID, delta int, reason string) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, errors.New("db error")
				}
			},
			wantTopic: LeaderboardManualPointAdjustmentFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleManualPointAdjustment(context.Background(), tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) > 0 && got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleRecalculateRound(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	roundID := sharedtypes.RoundID(uuid.New())
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)

	tests := []struct {
		name           string
		payload        *RecalculateRoundPayloadV1
		setupFake      func(*FakeService)
		roundLookup    RoundLookup
		wantTopic      string
	}{
		{
			name:    "success recalculation",
			payload: &RecalculateRoundPayloadV1{GuildID: guildID, RoundID: roundID},
			roundLookup: &FakeRoundLookup{
				GetRoundFunc: func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						Participants: []roundtypes.Participant{
							{UserID: "user-1", TagNumber: &tag1},
							{UserID: "user-2", TagNumber: &tag2},
						},
					}, nil
				},
			},
			setupFake: func(f *FakeService) {
				f.ProcessRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID, pr []leaderboardservice.PlayerResult, src sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
					return results.SuccessResult[leaderboardservice.ProcessRoundResult, error](leaderboardservice.ProcessRoundResult{
						LeaderboardData: leaderboardtypes.LeaderboardData{},
						PointsAwarded:   map[sharedtypes.DiscordID]int{"user-1": 100, "user-2": 0},
					}), nil
				}
			},
			wantTopic: LeaderboardRecalculateRoundSuccessV1,
		},
		{
			name:        "nil round lookup",
			payload:     &RecalculateRoundPayloadV1{GuildID: guildID, RoundID: roundID},
			roundLookup: nil,
			wantTopic:   LeaderboardRecalculateRoundFailedV1,
		},
		{
			name:    "round not found",
			payload: &RecalculateRoundPayloadV1{GuildID: guildID, RoundID: roundID},
			roundLookup: &FakeRoundLookup{
				GetRoundFunc: func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, nil
				},
			},
			wantTopic: LeaderboardRecalculateRoundFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{
				service:     fakeSvc,
				logger:      slog.Default(),
				roundLookup: tt.roundLookup,
			}

			got, err := h.HandleRecalculateRound(context.Background(), tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) > 0 && got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleStartNewSeason(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		payload   *StartNewSeasonPayloadV1
		setupFake func(*FakeService)
		wantTopic string
	}{
		{
			name:    "success",
			payload: &StartNewSeasonPayloadV1{GuildID: guildID, SeasonID: "2026-spring", SeasonName: "Spring 2026"},
			setupFake: func(f *FakeService) {
				f.StartNewSeasonFunc = func(ctx context.Context, g sharedtypes.GuildID, sid, sname string) (results.OperationResult[bool, error], error) {
					return results.SuccessResult[bool, error](true), nil
				}
			},
			wantTopic: LeaderboardStartNewSeasonSuccessV1,
		},
		{
			name:    "failure",
			payload: &StartNewSeasonPayloadV1{GuildID: guildID, SeasonID: "2026-spring", SeasonName: "Spring 2026"},
			setupFake: func(f *FakeService) {
				f.StartNewSeasonFunc = func(ctx context.Context, g sharedtypes.GuildID, sid, sname string) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, errors.New("duplicate season")
				}
			},
			wantTopic: LeaderboardStartNewSeasonFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleStartNewSeason(context.Background(), tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) > 0 && got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetSeasonStandings(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		payload   *GetSeasonStandingsPayloadV1
		setupFake func(*FakeService)
		wantTopic string
	}{
		{
			name:    "success",
			payload: &GetSeasonStandingsPayloadV1{GuildID: guildID, SeasonID: "default"},
			setupFake: func(f *FakeService) {
				f.GetSeasonStandingsForSeasonFunc = func(ctx context.Context, g sharedtypes.GuildID, sid string) (results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error], error) {
					return results.SuccessResult[[]leaderboardservice.SeasonStandingEntry, error]([]leaderboardservice.SeasonStandingEntry{
						{MemberID: "user-1", TotalPoints: 500, CurrentTier: "Gold"},
					}), nil
				}
			},
			wantTopic: LeaderboardGetSeasonStandingsResponseV1,
		},
		{
			name:    "failure",
			payload: &GetSeasonStandingsPayloadV1{GuildID: guildID, SeasonID: "invalid"},
			setupFake: func(f *FakeService) {
				f.GetSeasonStandingsForSeasonFunc = func(ctx context.Context, g sharedtypes.GuildID, sid string) (results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error], error) {
					return results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error]{}, errors.New("not found")
				}
			},
			wantTopic: LeaderboardGetSeasonStandingsFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleGetSeasonStandings(context.Background(), tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) > 0 && got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}
