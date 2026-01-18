package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// Service defines the interface for the round service.
type Service interface {
	// Create Round
	ValidateAndProcessRoundWithClock(ctx context.Context, payload roundevents.CreateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (results.OperationResult, error)
	ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface) (results.OperationResult, error)
	StoreRound(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundEntityCreatedPayloadV1) (results.OperationResult, error)
	UpdateRoundMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateAndProcessRoundUpdateWithClock(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (results.OperationResult, error)
	ValidateAndProcessRoundUpdate(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface) (results.OperationResult, error)
	UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayloadV1) (results.OperationResult, error)
	UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayloadV1) (results.OperationResult, error)

	// Delete Round
	ValidateRoundDeleteRequest(ctx context.Context, payload roundevents.RoundDeleteRequestPayloadV1) (results.OperationResult, error)
	DeleteRound(ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayloadV1) (results.OperationResult, error)

	// Start Round
	ProcessRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult, error)

	// Join Round
	ValidateParticipantJoinRequest(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)
	UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)
	ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayloadV1) (results.OperationResult, error)
	CheckParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)

	// Score Round
	ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayloadV1) (results.OperationResult, error)
	UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayloadV1) (results.OperationResult, error)
	UpdateParticipantScoresBulk(ctx context.Context, payload roundevents.ScoreBulkUpdateRequestPayloadV1) (results.OperationResult, error)
	CheckAllScoresSubmitted(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayloadV1) (results.OperationResult, error)

	// Finalize Round
	FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) (results.OperationResult, error)
	NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) (results.OperationResult, error)

	// Round Reminder
	ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayloadV1) (results.OperationResult, error)

	// Retrieve Round
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult, error)

	// Schedule Round Events
	ScheduleRoundEvents(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundScheduledPayloadV1, discordMessageID string) (results.OperationResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTags(ctx context.Context, guildID sharedtypes.GuildID, changedTags map[sharedtypes.DiscordID]sharedtypes.TagNumber) (results.OperationResult, error)

	// Scorecard Import
	CreateImportJob(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error)
	HandleScorecardURLRequested(ctx context.Context, payload roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error)
	ParseScorecard(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1, fileData []byte) (results.OperationResult, error)
	IngestParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (results.OperationResult, error)
	ApplyImportedScores(ctx context.Context, payload roundevents.ImportCompletedPayloadV1) (results.OperationResult, error)
}
