package leaderboardhandlers

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
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
var _ userservice.Service = (*FakeUserService)(nil)
var _ utils.Helpers = (*FakeHelpers)(nil)

// FakeUserService implements userservice.Service for handler testing.
type FakeUserService struct {
	LookupProfilesFunc func(ctx context.Context, userIDs []sharedtypes.DiscordID) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error)
}

func NewFakeUserService() *FakeUserService {
	return &FakeUserService{}
}

func (f *FakeUserService) LookupProfiles(ctx context.Context, userIDs []sharedtypes.DiscordID) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
	if f.LookupProfilesFunc != nil {
		return f.LookupProfilesFunc(ctx, userIDs)
	}
	// Return empty map results
	return results.SuccessResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error](map[sharedtypes.DiscordID]*usertypes.UserProfile{}), nil
}

// Implement other methods as no-ops to satisfy interface
func (f *FakeUserService) CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
	return userservice.UserResult{}, nil
}
func (f *FakeUserService) UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (userservice.UpdateIdentityResult, error) {
	return userservice.UpdateIdentityResult{}, nil
}
func (f *FakeUserService) GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserRoleResult, error) {
	return userservice.UserRoleResult{}, nil
}
func (f *FakeUserService) GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error) {
	return userservice.UserWithMembershipResult{}, nil
}
func (f *FakeUserService) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (userservice.UserWithMembershipResult, error) {
	return userservice.UserWithMembershipResult{}, nil
}
func (f *FakeUserService) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (userservice.UserWithMembershipResult, error) {
	return userservice.UserWithMembershipResult{}, nil
}
func (f *FakeUserService) UpdateUDiscIdentity(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (userservice.UpdateIdentityResult, error) {
	return userservice.UpdateIdentityResult{}, nil
}
func (f *FakeUserService) MatchParsedScorecard(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (userservice.MatchResultResult, error) {
	return userservice.MatchResultResult{}, nil
}
func (f *FakeUserService) UpdateUserProfile(ctx context.Context, userID sharedtypes.DiscordID, displayName, avatarHash string) error {
	return nil
}

// FakeHelpers implements utils.Helpers for testing
type FakeHelpers struct{}

func (f *FakeHelpers) CreateResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	return message.NewMessage("test-id", nil), nil
}

func (f *FakeHelpers) CreateNewMessage(payload interface{}, topic string) (*message.Message, error) {
	return message.NewMessage("test-id", nil), nil
}

func (f *FakeHelpers) UnmarshalPayload(msg *message.Message, payload interface{}) error {
	return nil
}
