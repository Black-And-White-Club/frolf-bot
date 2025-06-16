package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
)

// Service defines the interface for the round service.
type Service interface {
	// Create Round
	ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error)
	StoreRound(ctx context.Context, payload roundevents.RoundEntityCreatedPayload) (RoundOperationResult, error)
	UpdateRoundMessageID(ctx context.Context, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateAndProcessRoundUpdate(ctx context.Context, payload roundevents.UpdateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error)
	UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayload) (RoundOperationResult, error)
	UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayload) (RoundOperationResult, error)

	// Delete Round
	ValidateRoundDeleteRequest(ctx context.Context, payload roundevents.RoundDeleteRequestPayload) (RoundOperationResult, error)
	DeleteRound(ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayload) (RoundOperationResult, error)

	// Start Round
	ProcessRoundStart(ctx context.Context, payload roundevents.RoundStartedPayload) (RoundOperationResult, error)

	// Join Round
	ValidateParticipantJoinRequest(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error)
	UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error)
	ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayload) (RoundOperationResult, error)
	CheckParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error)

	// Score Round
	ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayload) (RoundOperationResult, error)
	UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayload) (RoundOperationResult, error)
	CheckAllScoresSubmitted(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayload) (RoundOperationResult, error)

	// Finalize Round
	FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayload) (RoundOperationResult, error)
	NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayload) (RoundOperationResult, error)

	// Round Reminder
	ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayload) (RoundOperationResult, error)

	// Retrieve Round
	GetRound(ctx context.Context, roundID sharedtypes.RoundID) (RoundOperationResult, error)

	// Schedule Round Events
	ScheduleRoundEvents(ctx context.Context, payload roundevents.RoundScheduledPayload, discordMessageID string) (RoundOperationResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTags(ctx context.Context, payload roundevents.ScheduledRoundTagUpdatePayload) (RoundOperationResult, error)
}
