package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// Service defines the interface for the round service.
type Service interface {
	// Create Round
	ValidateAndProcessRoundWithClock(ctx context.Context, payload roundevents.CreateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (RoundOperationResult, error)
	ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error)
	StoreRound(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundEntityCreatedPayloadV1) (RoundOperationResult, error)
	UpdateRoundMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateAndProcessRoundUpdateWithClock(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (RoundOperationResult, error)
	ValidateAndProcessRoundUpdate(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error)
	UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayloadV1) (RoundOperationResult, error)
	UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayloadV1) (RoundOperationResult, error)

	// Delete Round
	ValidateRoundDeleteRequest(ctx context.Context, payload roundevents.RoundDeleteRequestPayloadV1) (RoundOperationResult, error)
	DeleteRound(ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayloadV1) (RoundOperationResult, error)

	// Start Round
	ProcessRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (RoundOperationResult, error)

	// Join Round
	ValidateParticipantJoinRequest(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (RoundOperationResult, error)
	UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (RoundOperationResult, error)
	ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayloadV1) (RoundOperationResult, error)
	CheckParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (RoundOperationResult, error)

	// Score Round
	ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayloadV1) (RoundOperationResult, error)
	UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayloadV1) (RoundOperationResult, error)
	CheckAllScoresSubmitted(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayloadV1) (RoundOperationResult, error)

	// Finalize Round
	FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) (RoundOperationResult, error)
	NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) (RoundOperationResult, error)

	// Round Reminder
	ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayloadV1) (RoundOperationResult, error)

	// Retrieve Round
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (RoundOperationResult, error)

	// Schedule Round Events
	ScheduleRoundEvents(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundScheduledPayloadV1, discordMessageID string) (RoundOperationResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTags(ctx context.Context, payload roundevents.ScheduledRoundTagUpdatePayloadV1) (RoundOperationResult, error)

	// Scorecard Import
	CreateImportJob(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1) (RoundOperationResult, error)
	HandleScorecardURLRequested(ctx context.Context, payload roundevents.ScorecardURLRequestedPayloadV1) (RoundOperationResult, error)
	ParseScorecard(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1, fileData []byte) (RoundOperationResult, error)
	IngestParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (RoundOperationResult, error)
	ApplyImportedScores(ctx context.Context, payload roundevents.ImportCompletedPayloadV1) (RoundOperationResult, error)
}
