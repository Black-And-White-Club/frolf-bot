package leaderboardhandlers

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
)

// FakeService implements leaderboardservice.Service for handler testing.
type FakeService struct {
	trace []string

	// Function fields allow per-test behavior configuration
	ExecuteBatchTagAssignmentFunc func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)
	TagSwapRequestedFunc          func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)
	GetLeaderboardFunc            func(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error)
	GetTagByUserIDFunc            func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error)
	RoundGetTagByUserIDFunc       func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error)
	CheckTagAvailabilityFunc      func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error)
	EnsureGuildLeaderboardFunc    func(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error)
}

func NewFakeService() *FakeService {
	return &FakeService{trace: []string{}}
}

func (f *FakeService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeService) Trace() []string {
	return f.trace
}

// FakeSagaCoordinator implements the SagaCoordinator interface for testing.
type FakeSagaCoordinator struct {
	trace []string

	// CapturedIntents allows tests to inspect what was sent to the saga
	CapturedIntents []saga.SwapIntent

	// ProcessIntentFunc allows per-test behavior configuration (e.g., returning errors)
	ProcessIntentFunc func(ctx context.Context, intent saga.SwapIntent) error
}

func NewFakeSagaCoordinator() *FakeSagaCoordinator {
	return &FakeSagaCoordinator{
		trace:           []string{},
		CapturedIntents: []saga.SwapIntent{},
	}
}

func (f *FakeSagaCoordinator) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeSagaCoordinator) Trace() []string {
	return f.trace
}

func (f *FakeSagaCoordinator) ProcessIntent(ctx context.Context, intent saga.SwapIntent) error {
	f.record("ProcessIntent")
	f.CapturedIntents = append(f.CapturedIntents, intent)

	if f.ProcessIntentFunc != nil {
		return f.ProcessIntentFunc(ctx, intent)
	}
	return nil
}

// --- Interface Implementation ---
func (f *FakeService) ExecuteBatchTagAssignment(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	f.record("ExecuteBatchTagAssignment")
	if f.ExecuteBatchTagAssignmentFunc != nil {
		return f.ExecuteBatchTagAssignmentFunc(ctx, guildID, requests, updateID, source)
	}
	// Return empty domain data and nil error by default
	return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, nil
}

func (f *FakeService) TagSwapRequested(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	f.record("TagSwapRequested")
	if f.TagSwapRequestedFunc != nil {
		return f.TagSwapRequestedFunc(ctx, guildID, userID, targetTag)
	}
	return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, nil
}

func (f *FakeService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
	f.record("GetLeaderboard")
	if f.GetLeaderboardFunc != nil {
		return f.GetLeaderboardFunc(ctx, guildID)
	}
	return results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error]{}, nil
}

func (f *FakeService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
	f.record("GetTagByUserID")
	if f.GetTagByUserIDFunc != nil {
		return f.GetTagByUserIDFunc(ctx, guildID, userID)
	}
	return results.OperationResult[sharedtypes.TagNumber, error]{}, nil
}

func (f *FakeService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
	f.record("RoundGetTagByUserID")
	if f.RoundGetTagByUserIDFunc != nil {
		return f.RoundGetTagByUserIDFunc(ctx, guildID, userID)
	}
	return results.OperationResult[sharedtypes.TagNumber, error]{}, nil
}

func (f *FakeService) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error) {
	f.record("CheckTagAvailability")
	if f.CheckTagAvailabilityFunc != nil {
		return f.CheckTagAvailabilityFunc(ctx, guildID, userID, tagNumber)
	}
	return results.OperationResult[leaderboardservice.TagAvailabilityResult, error]{}, nil
}

func (f *FakeService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	f.record("EnsureGuildLeaderboard")
	if f.EnsureGuildLeaderboardFunc != nil {
		return f.EnsureGuildLeaderboardFunc(ctx, guildID)
	}
	return results.OperationResult[bool, error]{}, nil
}

// Ensure interface compliance
var _ leaderboardservice.Service = (*FakeService)(nil)
var _ saga.SagaCoordinator = (*FakeSagaCoordinator)(nil)
