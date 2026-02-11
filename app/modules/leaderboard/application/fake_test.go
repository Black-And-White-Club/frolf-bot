package leaderboardservice

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Leaderboard Repo
// ------------------------

type FakeLeaderboardRepo struct {
	trace []string

	// Points & Standings Stubs
	SavePointHistoryFunc     func(ctx context.Context, db bun.IDB, guildID string, history *leaderboarddb.PointHistory) error
	BulkSavePointHistoryFunc func(ctx context.Context, db bun.IDB, guildID string, histories []*leaderboarddb.PointHistory) error

	UpsertSeasonStandingFunc      func(ctx context.Context, db bun.IDB, guildID string, standing *leaderboarddb.SeasonStanding) error
	BulkUpsertSeasonStandingsFunc func(ctx context.Context, db bun.IDB, guildID string, standings []*leaderboarddb.SeasonStanding) error

	GetSeasonStandingFunc  func(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error)
	GetSeasonBestTagsFunc  func(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error)
	CountSeasonMembersFunc func(ctx context.Context, db bun.IDB, guildID string, seasonID string) (int, error)
	GetSeasonStandingsFunc func(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error)

	// Rollback Stubs
	GetPointHistoryForRoundFunc    func(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error)
	DeletePointHistoryForRoundFunc func(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) error
	DecrementSeasonStandingFunc    func(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error
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

func (f *FakeLeaderboardRepo) SavePointHistory(ctx context.Context, db bun.IDB, guildID string, history *leaderboarddb.PointHistory) error {
	f.record("SavePointHistory")
	if f.SavePointHistoryFunc != nil {
		return f.SavePointHistoryFunc(ctx, db, guildID, history)
	}
	return nil
}

func (f *FakeLeaderboardRepo) BulkSavePointHistory(ctx context.Context, db bun.IDB, guildID string, histories []*leaderboarddb.PointHistory) error {
	f.record("BulkSavePointHistory")
	if f.BulkSavePointHistoryFunc != nil {
		return f.BulkSavePointHistoryFunc(ctx, db, guildID, histories)
	}
	return nil
}

func (f *FakeLeaderboardRepo) UpsertSeasonStanding(ctx context.Context, db bun.IDB, guildID string, standing *leaderboarddb.SeasonStanding) error {
	f.record("UpsertSeasonStanding")
	if f.UpsertSeasonStandingFunc != nil {
		return f.UpsertSeasonStandingFunc(ctx, db, guildID, standing)
	}
	return nil
}

func (f *FakeLeaderboardRepo) BulkUpsertSeasonStandings(ctx context.Context, db bun.IDB, guildID string, standings []*leaderboarddb.SeasonStanding) error {
	f.record("BulkUpsertSeasonStandings")
	if f.BulkUpsertSeasonStandingsFunc != nil {
		return f.BulkUpsertSeasonStandingsFunc(ctx, db, guildID, standings)
	}
	return nil
}

func (f *FakeLeaderboardRepo) GetSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStanding")
	if f.GetSeasonStandingFunc != nil {
		return f.GetSeasonStandingFunc(ctx, db, guildID, memberID)
	}
	return nil, nil // Default to no standing
}

func (f *FakeLeaderboardRepo) GetSeasonBestTags(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
	f.record("GetSeasonBestTags")
	if f.GetSeasonBestTagsFunc != nil {
		return f.GetSeasonBestTagsFunc(ctx, db, guildID, seasonID, memberIDs)
	}
	return make(map[sharedtypes.DiscordID]int), nil
}

func (f *FakeLeaderboardRepo) CountSeasonMembers(ctx context.Context, db bun.IDB, guildID string, seasonID string) (int, error) {
	f.record("CountSeasonMembers")
	if f.CountSeasonMembersFunc != nil {
		return f.CountSeasonMembersFunc(ctx, db, guildID, seasonID)
	}
	return 0, nil
}

func (f *FakeLeaderboardRepo) GetSeasonStandings(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStandings")
	if f.GetSeasonStandingsFunc != nil {
		return f.GetSeasonStandingsFunc(ctx, db, guildID, seasonID, memberIDs)
	}
	return make(map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding), nil
}

func (f *FakeLeaderboardRepo) GetPointHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error) {
	f.record("GetPointHistoryForRound")
	if f.GetPointHistoryForRoundFunc != nil {
		return f.GetPointHistoryForRoundFunc(ctx, db, guildID, roundID)
	}
	return nil, nil // Default to no history
}

func (f *FakeLeaderboardRepo) DeletePointHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) error {
	f.record("DeletePointHistoryForRound")
	if f.DeletePointHistoryForRoundFunc != nil {
		return f.DeletePointHistoryForRoundFunc(ctx, db, guildID, roundID)
	}
	return nil
}

func (f *FakeLeaderboardRepo) DecrementSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error {
	f.record("DecrementSeasonStanding")
	if f.DecrementSeasonStandingFunc != nil {
		return f.DecrementSeasonStandingFunc(ctx, db, guildID, memberID, seasonID, pointsToRemove)
	}
	return nil
}

// --- Season Management ---

func (f *FakeLeaderboardRepo) GetActiveSeason(ctx context.Context, db bun.IDB, guildID string) (*leaderboarddb.Season, error) {
	f.record("GetActiveSeason")
	return &leaderboarddb.Season{GuildID: guildID, ID: "default", Name: "Default Season", IsActive: true}, nil
}

func (f *FakeLeaderboardRepo) CreateSeason(ctx context.Context, db bun.IDB, guildID string, season *leaderboarddb.Season) error {
	f.record("CreateSeason")
	return nil
}

func (f *FakeLeaderboardRepo) DeactivateAllSeasons(ctx context.Context, db bun.IDB, guildID string) error {
	f.record("DeactivateAllSeasons")
	return nil
}

func (f *FakeLeaderboardRepo) GetPointHistoryForMember(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, limit int) ([]leaderboarddb.PointHistory, error) {
	f.record("GetPointHistoryForMember")
	return nil, nil
}

func (f *FakeLeaderboardRepo) GetSeasonStandingsBySeasonID(ctx context.Context, db bun.IDB, guildID string, seasonID string) ([]leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStandingsBySeasonID")
	return nil, nil
}

// --- Accessors for assertions ---

func (f *FakeLeaderboardRepo) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// Ensure the fake actually satisfies the interface
var _ leaderboarddb.Repository = (*FakeLeaderboardRepo)(nil)

// ------------------------
// Fake Command Pipeline
// ------------------------

type FakeCommandPipeline struct {
	ProcessRoundFunc func(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error)
	ApplyTagsFunc    func(
		ctx context.Context,
		guildID string,
		requests []sharedtypes.TagAssignmentRequest,
		source sharedtypes.ServiceUpdateSource,
		updateID sharedtypes.RoundID,
	) (leaderboardtypes.LeaderboardData, error)
	StartSeasonFunc  func(ctx context.Context, guildID, seasonID, seasonName string) error
	EndSeasonFunc    func(ctx context.Context, guildID string) error
	ResetTagsFunc    func(ctx context.Context, guildID string, finishOrder []string) ([]leaderboarddomain.TagChange, error)
	GetTaggedFunc    func(ctx context.Context, guildID string) ([]TaggedMemberView, error)
	GetMemberTagFunc func(ctx context.Context, guildID, memberID string) (int, bool, error)
	CheckTagFunc     func(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error)
}

func (f *FakeCommandPipeline) ProcessRound(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
	if f.ProcessRoundFunc != nil {
		return f.ProcessRoundFunc(ctx, cmd)
	}
	return &ProcessRoundOutput{}, nil
}

func (f *FakeCommandPipeline) ApplyTagAssignments(
	ctx context.Context,
	guildID string,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	updateID sharedtypes.RoundID,
) (leaderboardtypes.LeaderboardData, error) {
	if f.ApplyTagsFunc != nil {
		return f.ApplyTagsFunc(ctx, guildID, requests, source, updateID)
	}
	return leaderboardtypes.LeaderboardData{}, nil
}

func (f *FakeCommandPipeline) StartSeason(ctx context.Context, guildID, seasonID, seasonName string) error {
	if f.StartSeasonFunc != nil {
		return f.StartSeasonFunc(ctx, guildID, seasonID, seasonName)
	}
	return nil
}

func (f *FakeCommandPipeline) EndSeason(ctx context.Context, guildID string) error {
	if f.EndSeasonFunc != nil {
		return f.EndSeasonFunc(ctx, guildID)
	}
	return nil
}

func (f *FakeCommandPipeline) ResetTags(ctx context.Context, guildID string, finishOrder []string) ([]leaderboarddomain.TagChange, error) {
	if f.ResetTagsFunc != nil {
		return f.ResetTagsFunc(ctx, guildID, finishOrder)
	}
	return nil, nil
}

func (f *FakeCommandPipeline) GetTaggedMembers(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
	if f.GetTaggedFunc != nil {
		return f.GetTaggedFunc(ctx, guildID)
	}
	return nil, nil
}

func (f *FakeCommandPipeline) GetMemberTag(ctx context.Context, guildID, memberID string) (int, bool, error) {
	if f.GetMemberTagFunc != nil {
		return f.GetMemberTagFunc(ctx, guildID, memberID)
	}
	return 0, false, nil
}

func (f *FakeCommandPipeline) CheckTagAvailability(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
	if f.CheckTagFunc != nil {
		return f.CheckTagFunc(ctx, guildID, memberID, tagNumber)
	}
	return true, "", nil
}

var _ CommandPipeline = (*FakeCommandPipeline)(nil)
