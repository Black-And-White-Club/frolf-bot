package leaderboardhandlers

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
)

// FakeService implements leaderboardservice.Service for handler testing.
type FakeService struct {
	trace []string

	// Function fields allow per-test behavior configuration
	ExecuteBatchTagAssignmentFunc func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (leaderboardtypes.LeaderboardData, error)
	TagSwapRequestedFunc          func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (leaderboardtypes.LeaderboardData, error)
	GetLeaderboardFunc            func(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardtypes.LeaderboardEntry, error)
	GetTagByUserIDFunc            func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error)
	RoundGetTagByUserIDFunc       func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error)
	CheckTagAvailabilityFunc      func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (leaderboardservice.TagAvailabilityResult, error)
	EnsureGuildLeaderboardFunc    func(ctx context.Context, guildID sharedtypes.GuildID) error
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
) (leaderboardtypes.LeaderboardData, error) {
	f.record("ExecuteBatchTagAssignment")
	if f.ExecuteBatchTagAssignmentFunc != nil {
		return f.ExecuteBatchTagAssignmentFunc(ctx, guildID, requests, updateID, source)
	}
	// Return empty domain data and nil error by default
	return leaderboardtypes.LeaderboardData{}, nil
}

func (f *FakeService) TagSwapRequested(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (leaderboardtypes.LeaderboardData, error) {
	f.record("TagSwapRequested")
	if f.TagSwapRequestedFunc != nil {
		return f.TagSwapRequestedFunc(ctx, guildID, userID, targetTag)
	}
	return leaderboardtypes.LeaderboardData{}, nil
}

func (f *FakeService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardtypes.LeaderboardEntry, error) {
	f.record("GetLeaderboard")
	if f.GetLeaderboardFunc != nil {
		return f.GetLeaderboardFunc(ctx, guildID)
	}
	return []leaderboardtypes.LeaderboardEntry{}, nil
}

func (f *FakeService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
	f.record("GetTagByUserID")
	if f.GetTagByUserIDFunc != nil {
		return f.GetTagByUserIDFunc(ctx, guildID, userID)
	}
	return 0, nil
}

func (f *FakeService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
	f.record("RoundGetTagByUserID")
	if f.RoundGetTagByUserIDFunc != nil {
		return f.RoundGetTagByUserIDFunc(ctx, guildID, userID)
	}
	return 0, nil
}

func (f *FakeService) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (leaderboardservice.TagAvailabilityResult, error) {
	f.record("CheckTagAvailability")
	if f.CheckTagAvailabilityFunc != nil {
		return f.CheckTagAvailabilityFunc(ctx, guildID, userID, tagNumber)
	}
	return leaderboardservice.TagAvailabilityResult{}, nil
}

func (f *FakeService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) error {
	f.record("EnsureGuildLeaderboard")
	if f.EnsureGuildLeaderboardFunc != nil {
		return f.EnsureGuildLeaderboardFunc(ctx, guildID)
	}
	return nil
}

// Ensure interface compliance
var _ leaderboardservice.Service = (*FakeService)(nil)
var _ saga.SagaCoordinator = (*FakeSagaCoordinator)(nil)
