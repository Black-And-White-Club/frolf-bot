package roundservice

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// Service defines the interface for the round service.
type Service interface {
	// Create Round
	ValidateRoundCreationWithClock(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (CreateRoundResult, error)
	StoreRound(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (CreateRoundResult, error)
	UpdateRoundMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateRoundUpdateWithClock(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (UpdateRoundResult, error)
	ValidateRoundUpdate(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface) (UpdateRoundResult, error)
	UpdateRoundEntity(ctx context.Context, req *roundtypes.UpdateRoundRequest) (UpdateRoundResult, error)
	UpdateScheduledRoundEvents(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (UpdateScheduledRoundEventsResult, error)

	// Delete Round
	ValidateRoundDeletion(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error)
	DeleteRound(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error)

	// Start Round
	StartRound(ctx context.Context, req *roundtypes.StartRoundRequest) (StartRoundResult, error)

	// Join Round
	ValidateJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error)
	JoinRound(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)
	CheckParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error)
	ValidateParticipantJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error)
	UpdateParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)
	ParticipantRemoval(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error)

	// Score Round
	ValidateScoreUpdateRequest(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error)
	UpdateParticipantScore(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (ScoreUpdateResult, error)
	UpdateParticipantScoresBulk(ctx context.Context, req *roundtypes.BulkScoreUpdateRequest) (BulkScoreUpdateResult, error)
	CheckAllScoresSubmitted(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (AllScoresSubmittedResult, error)

	// Finalize Round
	FinalizeRound(ctx context.Context, req *roundtypes.FinalizeRoundInput) (FinalizeRoundResult, error)
	NotifyScoreModule(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error)

	// Round Reminder
	ProcessRoundReminder(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (ProcessRoundReminderResult, error)

	// Retrieve Round
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error)
	GetRoundsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)

	// Schedule Round Events
	ScheduleRoundEvents(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (ScheduleRoundEventsResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTags(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (UpdateScheduledRoundsWithNewTagsResult, error)

	// Scorecard Import
	ScorecardURLRequested(ctx context.Context, req *roundtypes.ImportCreateJobInput) (CreateImportJobResult, error)
	CreateImportJob(ctx context.Context, req *roundtypes.ImportCreateJobInput) (CreateImportJobResult, error)
	ParseScorecard(ctx context.Context, req *roundtypes.ImportParseScorecardInput) (ParseScorecardResult, error)
	NormalizeParsedScorecard(ctx context.Context, data *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error)
	IngestNormalizedScorecard(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error)
	ApplyImportedScores(ctx context.Context, req roundtypes.ImportApplyScoresInput) (ApplyImportedScoresResult, error)
}

// =============================================================================
// DTOs (Data Transfer Objects)
// =============================================================================

// Result Aliases
type ApplyImportedScoresResult = results.OperationResult[*roundtypes.ImportApplyScoresResult, error]
type CreateRoundResult = results.OperationResult[*roundtypes.CreateRoundResult, error]
type FinalizeRoundResult = results.OperationResult[*roundtypes.FinalizeRoundResult, error]
type CreateImportJobResult = results.OperationResult[roundtypes.CreateImportJobResult, error]
type ParseScorecardResult = results.OperationResult[roundtypes.ParsedScorecard, error]
type UpdateScheduledRoundsWithNewTagsResult = results.OperationResult[*roundtypes.ScheduledRoundsSyncResult, error]
type UpdateScheduledRoundEventsResult = results.OperationResult[bool, error]
type UpdateRoundResult = results.OperationResult[*roundtypes.UpdateRoundResult, error]
type ProcessRoundReminderResult = results.OperationResult[roundtypes.ProcessRoundReminderResult, error]
type ScheduleRoundEventsResult = results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]
type ScoreUpdateResult = results.OperationResult[*roundtypes.ScoreUpdateResult, error]
type BulkScoreUpdateResult = results.OperationResult[*roundtypes.BulkScoreUpdateResult, error]
type AllScoresSubmittedResult = results.OperationResult[*roundtypes.AllScoresSubmittedResult, error]
type StartRoundResult = results.OperationResult[*roundtypes.Round, error]

type ProcessRoundReminderRequest struct {
	GuildID   sharedtypes.GuildID
	RoundID   sharedtypes.RoundID
	StartTime *sharedtypes.StartTime
	Title     *string
	Location  *string
}

// Import DTOs placeholders
