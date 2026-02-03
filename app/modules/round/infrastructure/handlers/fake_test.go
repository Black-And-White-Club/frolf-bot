package roundhandlers

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ------------------------
// Fake Service
// ------------------------

type FakeService struct {
	trace []string

	// Create Round
	ValidateRoundCreationWithClockFunc func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error)
	StoreRoundFunc                     func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error)
	UpdateRoundMessageIDFunc           func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateRoundUpdateWithClockFunc func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error)
	ValidateRoundUpdateFunc          func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface) (roundservice.UpdateRoundResult, error)
	UpdateRoundEntityFunc            func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error)
	UpdateScheduledRoundEventsFunc   func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error)

	// Delete Round
	ValidateRoundDeletionFunc func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error)
	DeleteRoundFunc           func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error)

	// Start Round
	StartRoundFunc func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error)

	// Join Round
	ValidateJoinRequestFunc            func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error)
	JoinRoundFunc                      func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)
	CheckParticipantStatusFunc         func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error)
	ValidateParticipantJoinRequestFunc func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error)
	UpdateParticipantStatusFunc        func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)
	ParticipantRemovalFunc             func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)

	// Score Round
	ValidateScoreUpdateRequestFunc  func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error)
	UpdateParticipantScoreFunc      func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error)
	UpdateParticipantScoresBulkFunc func(ctx context.Context, req *roundtypes.BulkScoreUpdateRequest) (roundservice.BulkScoreUpdateResult, error)
	CheckAllScoresSubmittedFunc     func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error)

	// Finalize Round
	FinalizeRoundFunc     func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error)
	NotifyScoreModuleFunc func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error)

	// Round Reminder
	ProcessRoundReminderFunc func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error)

	// Retrieve Round
	GetRoundFunc          func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error)
	GetRoundsForGuildFunc func(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)

	// Schedule Round Events
	ScheduleRoundEventsFunc func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTagsFunc func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error)

	// Scorecard Import
	ScorecardURLRequestedFunc     func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error)
	CreateImportJobFunc           func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error)
	ParseScorecardFunc            func(ctx context.Context, req *roundtypes.ImportParseScorecardInput) (roundservice.ParseScorecardResult, error)
	NormalizeParsedScorecardFunc  func(ctx context.Context, data *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error)
	IngestNormalizedScorecardFunc func(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error)
	ApplyImportedScoresFunc       func(ctx context.Context, req roundtypes.ImportApplyScoresInput) (roundservice.ApplyImportedScoresResult, error)
}

func NewFakeService() *FakeService {
	return &FakeService{trace: []string{}}
}

func (f *FakeService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// --- Implementation ---

// Create Round

func (f *FakeService) ValidateRoundCreationWithClock(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
	f.record("ValidateRoundCreationWithClock")
	if f.ValidateRoundCreationWithClockFunc != nil {
		return f.ValidateRoundCreationWithClockFunc(ctx, req, timeParser, clock)
	}
	return roundservice.CreateRoundResult{}, nil
}

func (f *FakeService) StoreRound(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
	f.record("StoreRound")
	if f.StoreRoundFunc != nil {
		return f.StoreRoundFunc(ctx, round, guildID)
	}
	return roundservice.CreateRoundResult{}, nil
}

func (f *FakeService) UpdateRoundMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
	f.record("UpdateRoundMessageID")
	if f.UpdateRoundMessageIDFunc != nil {
		return f.UpdateRoundMessageIDFunc(ctx, guildID, roundID, discordMessageID)
	}
	return nil, nil
}

// Update Round

func (f *FakeService) ValidateRoundUpdateWithClock(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error) {
	f.record("ValidateRoundUpdateWithClock")
	if f.ValidateRoundUpdateWithClockFunc != nil {
		return f.ValidateRoundUpdateWithClockFunc(ctx, req, timeParser, clock)
	}
	return roundservice.UpdateRoundResult{}, nil
}

func (f *FakeService) ValidateRoundUpdate(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface) (roundservice.UpdateRoundResult, error) {
	f.record("ValidateRoundUpdate")
	if f.ValidateRoundUpdateFunc != nil {
		return f.ValidateRoundUpdateFunc(ctx, req, timeParser)
	}
	return roundservice.UpdateRoundResult{}, nil
}

func (f *FakeService) UpdateRoundEntity(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
	f.record("UpdateRoundEntity")
	if f.UpdateRoundEntityFunc != nil {
		return f.UpdateRoundEntityFunc(ctx, req)
	}
	return roundservice.UpdateRoundResult{}, nil
}

func (f *FakeService) UpdateScheduledRoundEvents(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
	f.record("UpdateScheduledRoundEvents")
	if f.UpdateScheduledRoundEventsFunc != nil {
		return f.UpdateScheduledRoundEventsFunc(ctx, req)
	}
	return roundservice.UpdateScheduledRoundEventsResult{}, nil
}

// Delete Round

func (f *FakeService) ValidateRoundDeletion(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("ValidateRoundDeletion")
	if f.ValidateRoundDeletionFunc != nil {
		return f.ValidateRoundDeletionFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

func (f *FakeService) DeleteRound(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
	f.record("DeleteRound")
	if f.DeleteRoundFunc != nil {
		return f.DeleteRoundFunc(ctx, req)
	}
	return results.OperationResult[bool, error]{}, nil
}

// Start Round

func (f *FakeService) StartRound(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
	f.record("StartRound")
	if f.StartRoundFunc != nil {
		return f.StartRoundFunc(ctx, req)
	}
	return roundservice.StartRoundResult{}, nil
}

// Join Round

func (f *FakeService) ValidateJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error) {
	f.record("ValidateJoinRequest")
	if f.ValidateJoinRequestFunc != nil {
		return f.ValidateJoinRequestFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.JoinRoundRequest, error]{}, nil
}

func (f *FakeService) JoinRound(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("JoinRound")
	if f.JoinRoundFunc != nil {
		return f.JoinRoundFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

func (f *FakeService) CheckParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
	f.record("CheckParticipantStatus")
	if f.CheckParticipantStatusFunc != nil {
		return f.CheckParticipantStatusFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{}, nil
}

func (f *FakeService) ValidateParticipantJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error) {
	f.record("ValidateParticipantJoinRequest")
	if f.ValidateParticipantJoinRequestFunc != nil {
		return f.ValidateParticipantJoinRequestFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.JoinRoundRequest, error]{}, nil
}

func (f *FakeService) UpdateParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("UpdateParticipantStatus")
	if f.UpdateParticipantStatusFunc != nil {
		return f.UpdateParticipantStatusFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

func (f *FakeService) ParticipantRemoval(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("ParticipantRemoval")
	if f.ParticipantRemovalFunc != nil {
		return f.ParticipantRemovalFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

// Score Round

func (f *FakeService) ValidateScoreUpdateRequest(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
	f.record("ValidateScoreUpdateRequest")
	if f.ValidateScoreUpdateRequestFunc != nil {
		return f.ValidateScoreUpdateRequestFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{}, nil
}

func (f *FakeService) UpdateParticipantScore(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error) {
	f.record("UpdateParticipantScore")
	if f.UpdateParticipantScoreFunc != nil {
		return f.UpdateParticipantScoreFunc(ctx, req)
	}
	return roundservice.ScoreUpdateResult{}, nil
}

func (f *FakeService) UpdateParticipantScoresBulk(ctx context.Context, req *roundtypes.BulkScoreUpdateRequest) (roundservice.BulkScoreUpdateResult, error) {
	f.record("UpdateParticipantScoresBulk")
	if f.UpdateParticipantScoresBulkFunc != nil {
		return f.UpdateParticipantScoresBulkFunc(ctx, req)
	}
	return roundservice.BulkScoreUpdateResult{}, nil
}

func (f *FakeService) CheckAllScoresSubmitted(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
	f.record("CheckAllScoresSubmitted")
	if f.CheckAllScoresSubmittedFunc != nil {
		return f.CheckAllScoresSubmittedFunc(ctx, req)
	}
	return roundservice.AllScoresSubmittedResult{}, nil
}

// Finalize Round

func (f *FakeService) FinalizeRound(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
	f.record("FinalizeRound")
	if f.FinalizeRoundFunc != nil {
		return f.FinalizeRoundFunc(ctx, req)
	}
	return roundservice.FinalizeRoundResult{}, nil
}

func (f *FakeService) NotifyScoreModule(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("NotifyScoreModule")
	if f.NotifyScoreModuleFunc != nil {
		return f.NotifyScoreModuleFunc(ctx, result)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

// Round Reminder

func (f *FakeService) ProcessRoundReminder(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
	f.record("ProcessRoundReminder")
	if f.ProcessRoundReminderFunc != nil {
		return f.ProcessRoundReminderFunc(ctx, req)
	}
	return roundservice.ProcessRoundReminderResult{}, nil
}

// Retrieve Round

func (f *FakeService) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
	f.record("GetRound")
	if f.GetRoundFunc != nil {
		return f.GetRoundFunc(ctx, guildID, roundID)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

func (f *FakeService) GetRoundsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	f.record("GetRoundsForGuild")
	if f.GetRoundsForGuildFunc != nil {
		return f.GetRoundsForGuildFunc(ctx, guildID)
	}
	return []*roundtypes.Round{}, nil
}

// Schedule Round Events

func (f *FakeService) ScheduleRoundEvents(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
	f.record("ScheduleRoundEvents")
	if f.ScheduleRoundEventsFunc != nil {
		return f.ScheduleRoundEventsFunc(ctx, req)
	}
	return roundservice.ScheduleRoundEventsResult{}, nil
}

// Update Participant Tags

func (f *FakeService) UpdateScheduledRoundsWithNewTags(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
	f.record("UpdateScheduledRoundsWithNewTags")
	if f.UpdateScheduledRoundsWithNewTagsFunc != nil {
		return f.UpdateScheduledRoundsWithNewTagsFunc(ctx, req)
	}
	return roundservice.UpdateScheduledRoundsWithNewTagsResult{}, nil
}

// Scorecard Import

func (f *FakeService) ScorecardURLRequested(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
	f.record("ScorecardURLRequested")
	if f.ScorecardURLRequestedFunc != nil {
		return f.ScorecardURLRequestedFunc(ctx, req)
	}
	return roundservice.CreateImportJobResult{}, nil
}

func (f *FakeService) CreateImportJob(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
	f.record("CreateImportJob")
	if f.CreateImportJobFunc != nil {
		return f.CreateImportJobFunc(ctx, req)
	}
	return roundservice.CreateImportJobResult{}, nil
}

func (f *FakeService) ParseScorecard(ctx context.Context, req *roundtypes.ImportParseScorecardInput) (roundservice.ParseScorecardResult, error) {
	f.record("ParseScorecard")
	if f.ParseScorecardFunc != nil {
		return f.ParseScorecardFunc(ctx, req)
	}
	return roundservice.ParseScorecardResult{}, nil
}

func (f *FakeService) NormalizeParsedScorecard(ctx context.Context, data *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error) {
	f.record("NormalizeParsedScorecard")
	if f.NormalizeParsedScorecardFunc != nil {
		return f.NormalizeParsedScorecardFunc(ctx, data, meta)
	}
	return results.OperationResult[*roundtypes.NormalizedScorecard, error]{}, nil
}

func (f *FakeService) IngestNormalizedScorecard(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
	f.record("IngestNormalizedScorecard")
	if f.IngestNormalizedScorecardFunc != nil {
		return f.IngestNormalizedScorecardFunc(ctx, req)
	}
	return results.OperationResult[*roundtypes.IngestScorecardResult, error]{}, nil
}

func (f *FakeService) ApplyImportedScores(ctx context.Context, req roundtypes.ImportApplyScoresInput) (roundservice.ApplyImportedScoresResult, error) {
	f.record("ApplyImportedScores")
	if f.ApplyImportedScoresFunc != nil {
		return f.ApplyImportedScoresFunc(ctx, req)
	}
	return roundservice.ApplyImportedScoresResult{}, nil
}

var _ roundservice.Service = (*FakeService)(nil)
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
func (f *FakeUserService) UpdateUserProfile(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, displayName, avatarHash string) error {
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
