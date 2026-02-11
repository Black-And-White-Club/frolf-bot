package leaderboardservice

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_ProcessRound(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	roundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name          string
		playerResults []PlayerResult
		setupFake     func(*FakeLeaderboardRepo)
		expectErr     bool
		verify        func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], fake *FakeLeaderboardRepo)
	}{
		{
			name: "process round success - basic flow",
			playerResults: []PlayerResult{
				{"winner", 1},
				{"loser", 2},
			},
			setupFake: func(f *FakeLeaderboardRepo) {
				// Mock Active Leaderboard (for batch assignment)
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error) {
					return &leaderboardtypes.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: "winner", TagNumber: 2},
							{UserID: "loser", TagNumber: 1},
						},
					}, nil
				}
				// Mock Save Leaderboard
				f.SaveLeaderboardFunc = func(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error {
					return nil
				}

				// Mock Season Standings (Empty initially)
				f.GetSeasonStandingsFunc = func(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error) {
					return make(map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding), nil
				}

				// Mock Season Best Tags
				f.GetSeasonBestTagsFunc = func(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
					return map[sharedtypes.DiscordID]int{
						"winner": 2,
						"loser":  1,
					}, nil
				}

				// Mock Count Season Members
				f.CountSeasonMembersFunc = func(ctx context.Context, db bun.IDB, seasonID string) (int, error) {
					return 2, nil
				}
			},
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], fake *FakeLeaderboardRepo) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				data := res.Success.PointsAwarded
				// Winner (Tag 1) should beat Loser (Tag 2)
				// Winner was Tag 2, became Tag 1. Loser was Tag 1, became Tag 2.
				// Winner beats Loser.
				// Winner tier context: Best Tag 2 (Silver-ish? depends on count).
				// Count=2. Top 10% = 0.2 -> 1 person Gold (Tag 1). Next 30% -> 0.4 -> 1 person Silver?
				// Math.Ceil(2*0.1) = 1. Tag 1 is Gold.
				// Math.Ceil(2*0.4) = 1. Tag 1 is Gold.
				// Wait, bestTag logic update:
				// Winner: Old Best 2. New Tag 1. New Best 1. -> Gold
				// Loser: Old Best 1. New Tag 2. Best 1. -> Gold
				// Both Gold.
				// Winner (Gold) beats Loser (Gold).
				// Gold vs Gold -> Base Win (100) + No Bonus.
				// Points: Winner 100. Loser 0.

				if data["winner"] != 100 {
					t.Errorf("expected winner to get 100 points, got %d", data["winner"])
				}
			},
		},
		{
			name: "process round rollback triggered",
			playerResults: []PlayerResult{
				{"p1", 1},
			},
			setupFake: func(f *FakeLeaderboardRepo) {
				expectedSeasonID := "season-2025"
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error) {
					return &leaderboardtypes.Leaderboard{}, nil
				}
				// Ensure GetPointHistoryForRound is called
				f.GetPointHistoryForRoundFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error) {
					f.record("GetPointHistoryForRoundCalled")
					return []leaderboarddb.PointHistory{{ID: 1, Points: 50, MemberID: "p1", SeasonID: expectedSeasonID}}, nil
				}
				f.DeletePointHistoryForRoundFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) error {
					f.record("DeletePointHistoryForRoundCalled")
					return nil
				}
				f.DecrementSeasonStandingFunc = func(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error {
					if seasonID != expectedSeasonID {
						return fmt.Errorf("unexpected season on decrement: %s", seasonID)
					}
					f.record("DecrementSeasonStandingCalled")
					return nil
				}
				// Forward pass mocks
				f.SaveLeaderboardFunc = func(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error {
					f.record("SaveLeaderboard")
					return nil
				}
				f.CountSeasonMembersFunc = func(ctx context.Context, db bun.IDB, seasonID string) (int, error) {
					if seasonID != expectedSeasonID {
						return 0, fmt.Errorf("unexpected season on count: %s", seasonID)
					}
					return 1, nil
				}
				f.GetSeasonBestTagsFunc = func(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
					if seasonID != expectedSeasonID {
						return nil, fmt.Errorf("unexpected season on best tags: %s", seasonID)
					}
					return map[sharedtypes.DiscordID]int{"p1": 1}, nil
				}
				f.GetSeasonStandingsFunc = func(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error) {
					if seasonID != expectedSeasonID {
						return nil, fmt.Errorf("unexpected season on standings: %s", seasonID)
					}
					return make(map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding), nil
				}
				f.SavePointHistoryFunc = func(ctx context.Context, db bun.IDB, history *leaderboarddb.PointHistory) error {
					f.record("SavePointHistory")
					return nil
				}
				f.UpsertSeasonStandingFunc = func(ctx context.Context, db bun.IDB, standing *leaderboarddb.SeasonStanding) error {
					f.record("UpsertSeasonStanding")
					return nil
				}
			},
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundResult, error], fake *FakeLeaderboardRepo) {
				trace := fake.Trace()
				foundRollback := false
				foundForwardPass := false

				for _, call := range trace {
					if call == "DeletePointHistoryForRoundCalled" {
						foundRollback = true
					}
					if call == "SaveLeaderboard" {
						foundForwardPass = true
					}
				}
				if !foundRollback {
					t.Error("expected rollback to occur, but DeletePointHistoryForRound was not called")
				}
				if !foundForwardPass {
					t.Error("expected forward pass to occur after rollback, but SaveLeaderboard was not called")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &LeaderboardService{
				repo:    fakeRepo,
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			// We need a dummy context just for execution
			// Using ServiceUpdateSourceProcessScores
			res, err := s.ProcessRound(context.Background(), guildID, roundID, tt.playerResults, sharedtypes.ServiceUpdateSourceProcessScores)

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, fakeRepo)
			}
		})
	}
}
