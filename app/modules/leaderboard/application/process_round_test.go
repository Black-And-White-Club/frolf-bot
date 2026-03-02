package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_ProcessRound(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	roundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name          string
		playerResults []PlayerResult
		setupPipeline func(*FakeCommandPipeline)
		expectErr     bool
		verify        func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], err error)
	}{
		{
			name: "process round success - basic flow",
			playerResults: []PlayerResult{
				{PlayerID: "winner", TagNumber: 2},
				{PlayerID: "loser", TagNumber: 1},
			},
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ProcessRoundFunc = func(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
					if len(cmd.Participants) != 2 {
						t.Fatalf("expected 2 participants, got %d", len(cmd.Participants))
					}
					if cmd.Participants[0].MemberID != "loser" || cmd.Participants[0].FinishRank != 1 {
						t.Fatalf("unexpected participant ordering at position 0: %+v", cmd.Participants[0])
					}
					if cmd.Participants[1].MemberID != "winner" || cmd.Participants[1].FinishRank != 2 {
						t.Fatalf("unexpected participant ordering at position 1: %+v", cmd.Participants[1])
					}
					return &ProcessRoundOutput{
						FinalParticipantTags: map[string]int{
							"winner": 1,
							"loser":  2,
						},
						PointAwards: []leaderboarddomain.PointAward{
							{MemberID: "winner", Points: 100},
						},
					}, nil
				}
			},
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], err error) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				if res.Success.PointsAwarded["winner"] != 100 {
					t.Errorf("expected winner to get 100 points, got %d", res.Success.PointsAwarded["winner"])
				}
				if len(res.Success.LeaderboardData) != 2 {
					t.Fatalf("expected 2 leaderboard entries, got %d", len(res.Success.LeaderboardData))
				}
				if res.Success.LeaderboardData[0].TagNumber != 1 || res.Success.LeaderboardData[0].UserID != "winner" {
					t.Fatalf("unexpected leaderboard entry[0]: %+v", res.Success.LeaderboardData[0])
				}
			},
		},
		{
			name: "pipeline error bubbles up",
			playerResults: []PlayerResult{
				{PlayerID: "p1", TagNumber: 1},
			},
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ProcessRoundFunc = func(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
					return nil, errors.New("round processing failed")
				}
			},
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], err error) {
				if err == nil || err.Error() == "" {
					t.Fatal("expected non-empty error")
				}
			},
		},
		{
			name: "nil pipeline output returns error",
			playerResults: []PlayerResult{
				{PlayerID: "p1", TagNumber: 1},
			},
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ProcessRoundFunc = func(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
					return nil, nil
				}
			},
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], err error) {
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		// Two players sharing the same TagNumber must receive equal FinishRanks
		// (competition style: 1,1,3 not 1,2,3). Tags are unique in normal play,
		// but legacy data or admin recalc may encounter duplicates.
		{
			name: "tied tag numbers produce equal finish ranks (competition style)",
			playerResults: []PlayerResult{
				{PlayerID: "player-a", TagNumber: 5},
				{PlayerID: "player-b", TagNumber: 5},
				{PlayerID: "player-c", TagNumber: 6},
			},
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ProcessRoundFunc = func(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
					rankByMember := make(map[string]int, len(cmd.Participants))
					for _, participant := range cmd.Participants {
						rankByMember[participant.MemberID] = participant.FinishRank
					}
					if rankByMember["player-a"] != rankByMember["player-b"] {
						t.Errorf("tied tag players must share FinishRank: player-a=%d player-b=%d",
							rankByMember["player-a"], rankByMember["player-b"])
					}
					if rankByMember["player-a"] != 1 {
						t.Errorf("tied tag-5 players must have FinishRank 1, got %d", rankByMember["player-a"])
					}
					if rankByMember["player-c"] != 3 {
						t.Errorf("player-c (tag 6) must have FinishRank 3 (competition style), got %d", rankByMember["player-c"])
					}
					return &ProcessRoundOutput{
						FinalParticipantTags: map[string]int{"player-a": 5, "player-b": 6, "player-c": 7},
						PointAwards:          nil,
					}, nil
				}
			},
			expectErr: false,
			verify:    func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], err error) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			pipeline := &FakeCommandPipeline{}
			if tt.setupPipeline != nil {
				tt.setupPipeline(pipeline)
			}

			s := &LeaderboardService{
				repo:    fakeRepo,
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			s.SetCommandPipeline(pipeline)

			res, err := s.ProcessRound(context.Background(), guildID, roundID, tt.playerResults, sharedtypes.ServiceUpdateSourceProcessScores)

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err)
			}
		})
	}
}
