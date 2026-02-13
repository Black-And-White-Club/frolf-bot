package saga

import (
	"context"
	"errors"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	"github.com/nats-io/nats.go/jetstream"
)

// ------------------------
// Fake KeyValue
// ------------------------

type FakeKeyValue struct {
	jetstream.KeyValue // Embed to satisfy interface
	data               map[string][]byte
	trace              []string
}

func NewFakeKeyValue() *FakeKeyValue {
	return &FakeKeyValue{
		data:  make(map[string][]byte),
		trace: []string{},
	}
}

func (f *FakeKeyValue) Put(ctx context.Context, key string, value []byte) (uint64, error) {
	f.trace = append(f.trace, "Put")
	f.data[key] = value
	return 0, nil
}

func (f *FakeKeyValue) Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error) {
	f.trace = append(f.trace, "Get")
	val, ok := f.data[key]
	if !ok {
		return nil, jetstream.ErrKeyNotFound
	}
	return &FakeKeyValueEntry{value: val, key: key}, nil
}

func (f *FakeKeyValue) Delete(ctx context.Context, key string, opts ...jetstream.KVDeleteOpt) error {
	f.trace = append(f.trace, "Delete")
	delete(f.data, key)
	return nil
}

func (f *FakeKeyValue) Keys(ctx context.Context, opts ...jetstream.WatchOpt) ([]string, error) {
	f.trace = append(f.trace, "Keys")
	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return nil, jetstream.ErrNoKeysFound
	}
	return keys, nil
}

type FakeKeyValueEntry struct {
	jetstream.KeyValueEntry
	value []byte
	key   string
}

func (f *FakeKeyValueEntry) Value() []byte { return f.value }
func (f *FakeKeyValueEntry) Key() string   { return f.key }

// ------------------------
// Fake Leaderboard Service
// ------------------------

type FakeLeaderboardService struct {
	trace []string

	ExecuteBatchTagAssignmentFunc func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)
}

func (f *FakeLeaderboardService) ExecuteBatchTagAssignment(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	f.trace = append(f.trace, "ExecuteBatchTagAssignment")
	if f.ExecuteBatchTagAssignmentFunc != nil {
		return f.ExecuteBatchTagAssignmentFunc(ctx, guildID, requests, updateID, source)
	}
	return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{}), nil
}

// Implement other methods to satisfy interface (even if not used by saga)
func (f *FakeLeaderboardService) TagSwapRequested(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	return results.FailureResult[leaderboardtypes.LeaderboardData, error](errors.New("not implemented")), nil
}
func (f *FakeLeaderboardService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
	return results.FailureResult[[]leaderboardtypes.LeaderboardEntry, error](errors.New("not implemented")), nil
}
func (f *FakeLeaderboardService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
	return results.FailureResult[sharedtypes.TagNumber, error](errors.New("not implemented")), nil
}
func (f *FakeLeaderboardService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
	return results.FailureResult[sharedtypes.TagNumber, error](errors.New("not implemented")), nil
}
func (f *FakeLeaderboardService) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error) {
	return results.FailureResult[leaderboardservice.TagAvailabilityResult, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	return results.FailureResult[bool, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) ProcessRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	playerResults []leaderboardservice.PlayerResult,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
	return results.FailureResult[leaderboardservice.ProcessRoundResult, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) GetPointHistoryForMember(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, limit int) (results.OperationResult[[]leaderboardservice.PointHistoryEntry, error], error) {
	return results.FailureResult[[]leaderboardservice.PointHistoryEntry, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) AdjustPoints(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, pointsDelta int, reason string) (results.OperationResult[bool, error], error) {
	return results.FailureResult[bool, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) StartNewSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string, seasonName string) (results.OperationResult[bool, error], error) {
	return results.FailureResult[bool, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) GetSeasonStandingsForSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error], error) {
	return results.FailureResult[[]leaderboardservice.SeasonStandingEntry, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) ProcessRoundCommand(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
	return nil, leaderboardservice.ErrCommandPipelineUnavailable
}

func (f *FakeLeaderboardService) ResetTagsFromQualifyingRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	finishOrder []sharedtypes.DiscordID,
) ([]leaderboarddomain.TagChange, error) {
	return nil, leaderboardservice.ErrCommandPipelineUnavailable
}

func (f *FakeLeaderboardService) EndSeason(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	return results.SuccessResult[bool, error](true), nil
}

func (f *FakeLeaderboardService) GetTagHistory(ctx context.Context, guildID sharedtypes.GuildID, memberID string, limit int) ([]leaderboardservice.TagHistoryView, error) {
	f.trace = append(f.trace, "GetTagHistory")
	return nil, nil
}

func (f *FakeLeaderboardService) GetTagList(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardservice.TaggedMemberView, error) {
	f.trace = append(f.trace, "GetTagList")
	return nil, nil
}

func (f *FakeLeaderboardService) GenerateTagGraphPNG(ctx context.Context, guildID sharedtypes.GuildID, memberID string) ([]byte, error) {
	f.trace = append(f.trace, "GenerateTagGraphPNG")
	return nil, nil
}

func (f *FakeLeaderboardService) ListSeasons(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[[]leaderboardservice.SeasonInfo, error], error) {
	return results.FailureResult[[]leaderboardservice.SeasonInfo, error](errors.New("not implemented")), nil
}

func (f *FakeLeaderboardService) GetSeasonName(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (string, error) {
	return "", errors.New("not implemented")
}
