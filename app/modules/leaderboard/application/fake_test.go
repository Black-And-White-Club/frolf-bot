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

	GetActiveLeaderboardFunc  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error)
	SaveLeaderboardFunc       func(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error
	DeactivateLeaderboardFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error

	// Points & Standings Stubs
	SavePointHistoryFunc     func(ctx context.Context, db bun.IDB, history *leaderboarddb.PointHistory) error
	UpsertSeasonStandingFunc func(ctx context.Context, db bun.IDB, standing *leaderboarddb.SeasonStanding) error
	GetSeasonStandingFunc    func(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error)
	GetSeasonBestTagsFunc    func(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error)
	CountSeasonMembersFunc   func(ctx context.Context, db bun.IDB) (int, error)
	GetSeasonStandingsFunc   func(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error)

	// Rollback Stubs
	GetPointHistoryForRoundFunc    func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error)
	DeletePointHistoryForRoundFunc func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) error
	DecrementSeasonStandingFunc    func(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, pointsToRemove int) error
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

func (f *FakeLeaderboardRepo) GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error) {
	f.record("GetActiveLeaderboard")
	if f.GetActiveLeaderboardFunc != nil {
		return f.GetActiveLeaderboardFunc(ctx, db, guildID)
	}
	// Default: Return nil/NotFound style if not configured
	return nil, leaderboarddb.ErrNoActiveLeaderboard
}

func (f *FakeLeaderboardRepo) SaveLeaderboard(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error {
	f.record("SaveLeaderboard")
	if f.SaveLeaderboardFunc != nil {
		return f.SaveLeaderboardFunc(ctx, db, leaderboard)
	}
	return nil
}

func (f *FakeLeaderboardRepo) DeactivateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, id int64) error {
	f.record("DeactivateLeaderboard")
	if f.DeactivateLeaderboardFunc != nil {
		return f.DeactivateLeaderboardFunc(ctx, db, guildID, id)
	}
	return nil
}

func (f *FakeLeaderboardRepo) SavePointHistory(ctx context.Context, db bun.IDB, history *leaderboarddb.PointHistory) error {
	f.record("SavePointHistory")
	if f.SavePointHistoryFunc != nil {
		return f.SavePointHistoryFunc(ctx, db, history)
	}
	return nil
}

func (f *FakeLeaderboardRepo) UpsertSeasonStanding(ctx context.Context, db bun.IDB, standing *leaderboarddb.SeasonStanding) error {
	f.record("UpsertSeasonStanding")
	if f.UpsertSeasonStandingFunc != nil {
		return f.UpsertSeasonStandingFunc(ctx, db, standing)
	}
	return nil
}

func (f *FakeLeaderboardRepo) GetSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStanding")
	if f.GetSeasonStandingFunc != nil {
		return f.GetSeasonStandingFunc(ctx, db, memberID)
	}
	return nil, nil // Default to no standing
}

func (f *FakeLeaderboardRepo) GetSeasonBestTags(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
	f.record("GetSeasonBestTags")
	if f.GetSeasonBestTagsFunc != nil {
		return f.GetSeasonBestTagsFunc(ctx, db, memberIDs)
	}
	return make(map[sharedtypes.DiscordID]int), nil
}

func (f *FakeLeaderboardRepo) CountSeasonMembers(ctx context.Context, db bun.IDB) (int, error) {
	f.record("CountSeasonMembers")
	if f.CountSeasonMembersFunc != nil {
		return f.CountSeasonMembersFunc(ctx, db)
	}
	return 0, nil
}

func (f *FakeLeaderboardRepo) GetSeasonStandings(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStandings")
	if f.GetSeasonStandingsFunc != nil {
		return f.GetSeasonStandingsFunc(ctx, db, memberIDs)
	}
	return make(map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding), nil
}

func (f *FakeLeaderboardRepo) GetPointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error) {
	f.record("GetPointHistoryForRound")
	if f.GetPointHistoryForRoundFunc != nil {
		return f.GetPointHistoryForRoundFunc(ctx, db, roundID)
	}
	return nil, nil // Default to no history
}

func (f *FakeLeaderboardRepo) DeletePointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) error {
	f.record("DeletePointHistoryForRound")
	if f.DeletePointHistoryForRoundFunc != nil {
		return f.DeletePointHistoryForRoundFunc(ctx, db, roundID)
	}
	return nil
}

func (f *FakeLeaderboardRepo) DecrementSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, pointsToRemove int) error {
	f.record("DecrementSeasonStanding")
	if f.DecrementSeasonStandingFunc != nil {
		return f.DecrementSeasonStandingFunc(ctx, db, memberID, pointsToRemove)
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
