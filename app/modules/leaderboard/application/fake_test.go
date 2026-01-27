package leaderboardservice

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Leaderboard Repo
// ------------------------

type FakeLeaderboardRepo struct {
	trace []string

	GetActiveLeaderboardFunc  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error)
	UpdateLeaderboardFunc     func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error)
	CreateLeaderboardFunc     func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboard *leaderboarddb.Leaderboard) (*leaderboarddb.Leaderboard, error)
	DeactivateLeaderboardFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error
}

func NewFakeLeaderboardRepo() *FakeLeaderboardRepo {
	return &FakeLeaderboardRepo{
		trace: []string{},
	}
}

func (f *FakeLeaderboardRepo) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Repository Interface Implementation ---

func (f *FakeLeaderboardRepo) GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
	f.record("GetActiveLeaderboard")
	if f.GetActiveLeaderboardFunc != nil {
		return f.GetActiveLeaderboardFunc(ctx, db, guildID)
	}
	// Default: Return nil/NotFound style if not configured
	return nil, leaderboarddb.ErrNoActiveLeaderboard
}

func (f *FakeLeaderboardRepo) UpdateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error) {
	f.record("UpdateLeaderboard")
	if f.UpdateLeaderboardFunc != nil {
		return f.UpdateLeaderboardFunc(ctx, db, guildID, data, updateID, source)
	}
	return &leaderboarddb.Leaderboard{
		LeaderboardData: data,
		GuildID:         guildID,
		UpdateID:        updateID,
		IsActive:        true,
	}, nil
}

func (f *FakeLeaderboardRepo) CreateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, lb *leaderboarddb.Leaderboard) (*leaderboarddb.Leaderboard, error) {
	f.record("CreateLeaderboard")
	if f.CreateLeaderboardFunc != nil {
		return f.CreateLeaderboardFunc(ctx, db, guildID, lb)
	}
	lb.GuildID = guildID
	return lb, nil
}

func (f *FakeLeaderboardRepo) DeactivateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, id int64) error {
	f.record("DeactivateLeaderboard")
	if f.DeactivateLeaderboardFunc != nil {
		return f.DeactivateLeaderboardFunc(ctx, db, guildID, id)
	}
	return nil
}

// --- Accessors for assertions ---

func (f *FakeLeaderboardRepo) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// Ensure the fake actually satisfies the interface
var _ leaderboarddb.Repository = (*FakeLeaderboardRepo)(nil)
