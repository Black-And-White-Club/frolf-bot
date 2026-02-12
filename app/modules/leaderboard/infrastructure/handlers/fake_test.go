package leaderboardhandlers

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// FakeService implements leaderboardservice.Service for handler testing.
type FakeService struct {
	trace []string

	// Function fields allow per-test behavior configuration
	ExecuteBatchTagAssignmentFunc func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)
	TagSwapRequestedFunc          func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, targetTag sharedtypes.TagNumber) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)
	GetLeaderboardFunc            func(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error)
	GetTagByUserIDFunc            func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error)

	RoundGetTagByUserIDFunc          func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error)
	CheckTagAvailabilityFunc         func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error)
	EnsureGuildLeaderboardFunc       func(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error)
	ProcessRoundFunc                 func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, playerResults []leaderboardservice.PlayerResult, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error)
	ProcessRoundCommandFunc          func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error)
	ResetTagsFromQualifyingRoundFunc func(ctx context.Context, guildID sharedtypes.GuildID, finishOrder []sharedtypes.DiscordID) ([]leaderboarddomain.TagChange, error)
	EndSeasonFunc                    func(ctx context.Context, guildID sharedtypes.GuildID) error

	// Admin Operations
	GetPointHistoryForMemberFunc    func(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, limit int) (results.OperationResult[[]leaderboardservice.PointHistoryEntry, error], error)
	AdjustPointsFunc                func(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, pointsDelta int, reason string) (results.OperationResult[bool, error], error)
	StartNewSeasonFunc              func(ctx context.Context, guildID sharedtypes.GuildID, seasonID string, seasonName string) (results.OperationResult[bool, error], error)
	GetSeasonStandingsForSeasonFunc func(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error], error)
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

func (f *FakeService) ProcessRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	playerResults []leaderboardservice.PlayerResult,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
	f.record("ProcessRound")
	if f.ProcessRoundFunc != nil {
		return f.ProcessRoundFunc(ctx, guildID, roundID, playerResults, source)
	}
	return results.OperationResult[leaderboardservice.ProcessRoundResult, error]{}, nil
}

func (f *FakeService) ProcessRoundCommand(
	ctx context.Context,
	cmd leaderboardservice.ProcessRoundCommand,
) (*leaderboardservice.ProcessRoundOutput, error) {
	f.record("ProcessRoundCommand")
	if f.ProcessRoundCommandFunc != nil {
		return f.ProcessRoundCommandFunc(ctx, cmd)
	}
	return nil, leaderboardservice.ErrCommandPipelineUnavailable
}

func (f *FakeService) ResetTagsFromQualifyingRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	finishOrder []sharedtypes.DiscordID,
) ([]leaderboarddomain.TagChange, error) {
	f.record("ResetTagsFromQualifyingRound")
	if f.ResetTagsFromQualifyingRoundFunc != nil {
		return f.ResetTagsFromQualifyingRoundFunc(ctx, guildID, finishOrder)
	}
	return nil, leaderboardservice.ErrCommandPipelineUnavailable
}

func (f *FakeService) EndSeason(ctx context.Context, guildID sharedtypes.GuildID) error {
	f.record("EndSeason")
	if f.EndSeasonFunc != nil {
		return f.EndSeasonFunc(ctx, guildID)
	}
	return nil
}

func (f *FakeService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	f.record("EnsureGuildLeaderboard")
	if f.EnsureGuildLeaderboardFunc != nil {
		return f.EnsureGuildLeaderboardFunc(ctx, guildID)
	}
	return results.OperationResult[bool, error]{}, nil
}

// --- Admin Operations ---

func (f *FakeService) GetPointHistoryForMember(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, limit int) (results.OperationResult[[]leaderboardservice.PointHistoryEntry, error], error) {
	f.record("GetPointHistoryForMember")
	if f.GetPointHistoryForMemberFunc != nil {
		return f.GetPointHistoryForMemberFunc(ctx, guildID, memberID, limit)
	}
	return results.OperationResult[[]leaderboardservice.PointHistoryEntry, error]{}, nil
}

func (f *FakeService) AdjustPoints(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, pointsDelta int, reason string) (results.OperationResult[bool, error], error) {
	f.record("AdjustPoints")
	if f.AdjustPointsFunc != nil {
		return f.AdjustPointsFunc(ctx, guildID, memberID, pointsDelta, reason)
	}
	return results.OperationResult[bool, error]{}, nil
}

func (f *FakeService) StartNewSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string, seasonName string) (results.OperationResult[bool, error], error) {
	f.record("StartNewSeason")
	if f.StartNewSeasonFunc != nil {
		return f.StartNewSeasonFunc(ctx, guildID, seasonID, seasonName)
	}
	return results.OperationResult[bool, error]{}, nil
}

func (f *FakeService) GetSeasonStandingsForSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error], error) {
	f.record("GetSeasonStandingsForSeason")
	if f.GetSeasonStandingsForSeasonFunc != nil {
		return f.GetSeasonStandingsForSeasonFunc(ctx, guildID, seasonID)
	}
	return results.OperationResult[[]leaderboardservice.SeasonStandingEntry, error]{}, nil
}

// Ensure interface compliance
var _ leaderboardservice.Service = (*FakeService)(nil)
var _ saga.SagaCoordinator = (*FakeSagaCoordinator)(nil)
var _ userservice.Service = (*FakeUserService)(nil)
var _ utils.Helpers = (*FakeHelpers)(nil)

// FakeUserService implements userservice.Service for handler testing.
type FakeUserService struct {
	LookupProfilesFunc              func(ctx context.Context, userIDs []sharedtypes.DiscordID, guildID sharedtypes.GuildID) (results.OperationResult[*userservice.LookupProfilesResponse, error], error)
	GetClubUUIDByDiscordGuildIDFunc func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error)
}

func NewFakeUserService() *FakeUserService {
	return &FakeUserService{}
}

func (f *FakeUserService) LookupProfiles(ctx context.Context, userIDs []sharedtypes.DiscordID, guildID sharedtypes.GuildID) (results.OperationResult[*userservice.LookupProfilesResponse, error], error) {
	if f.LookupProfilesFunc != nil {
		return f.LookupProfilesFunc(ctx, userIDs, guildID)
	}
	// Return empty map results
	return results.SuccessResult[*userservice.LookupProfilesResponse, error](&userservice.LookupProfilesResponse{
		Profiles: make(map[sharedtypes.DiscordID]*usertypes.UserProfile),
	}), nil
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
func (f *FakeUserService) UpdateUserProfile(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, displayName, avatarHash string) error {
	return nil
}

func (f *FakeUserService) GetUUIDByDiscordID(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (f *FakeUserService) GetClubUUIDByDiscordGuildID(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	if f.GetClubUUIDByDiscordGuildIDFunc != nil {
		return f.GetClubUUIDByDiscordGuildIDFunc(ctx, guildID)
	}
	return uuid.New(), nil
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

// FakeRoundLookup implements RoundLookup for handler testing.
type FakeRoundLookup struct {
	GetRoundFunc func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
}

func (f *FakeRoundLookup) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	if f == nil || f.GetRoundFunc == nil {
		return nil, nil
	}
	return f.GetRoundFunc(ctx, guildID, roundID)
}
