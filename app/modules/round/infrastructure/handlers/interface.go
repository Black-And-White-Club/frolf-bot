package roundhandlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

type Handlers interface {
	// Round creation handlers
	HandleCreateRoundRequest(ctx context.Context, payload *roundevents.CreateRoundRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundEntityCreated(ctx context.Context, payload *roundevents.RoundEntityCreatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundEventMessageIDUpdate(ctx context.Context, payload *roundevents.RoundMessageIDUpdatePayloadV1) ([]handlerwrapper.Result, error)

	// Round deletion handlers
	HandleRoundDeleteRequest(ctx context.Context, payload *roundevents.RoundDeleteRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundDeleteValidated(ctx context.Context, payload *roundevents.RoundDeleteValidatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundDeleteAuthorized(ctx context.Context, payload *roundevents.RoundDeleteAuthorizedPayloadV1) ([]handlerwrapper.Result, error)

	// Round update handlers
	HandleRoundUpdateRequest(ctx context.Context, payload *roundevents.UpdateRoundRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundUpdateValidated(ctx context.Context, payload *roundevents.RoundUpdateValidatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundScheduleUpdate(ctx context.Context, payload *roundevents.RoundEntityUpdatedPayloadV1) ([]handlerwrapper.Result, error)

	// Score update handlers
	HandleScoreUpdateRequest(ctx context.Context, payload *roundevents.ScoreUpdateRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateValidated(ctx context.Context, payload *roundevents.ScoreUpdateValidatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantScoreUpdated(ctx context.Context, payload *roundevents.ParticipantScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error)

	// Round finalization handlers
	HandleAllScoresSubmitted(ctx context.Context, payload *roundevents.AllScoresSubmittedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedPayloadV1) ([]handlerwrapper.Result, error)

	// Round start handler
	HandleRoundStarted(ctx context.Context, payload *roundevents.RoundStartedPayloadV1) ([]handlerwrapper.Result, error)

	// Participant status handlers
	HandleParticipantJoinRequest(ctx context.Context, payload *roundevents.ParticipantJoinRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantJoinValidationRequest(ctx context.Context, payload *roundevents.ParticipantJoinValidationRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantStatusUpdateRequest(ctx context.Context, payload *roundevents.ParticipantJoinRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantRemovalRequest(ctx context.Context, payload *roundevents.ParticipantRemovalRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantDeclined(ctx context.Context, payload *roundevents.ParticipantDeclinedPayloadV1) ([]handlerwrapper.Result, error)

	// Tag lookup handlers
	HandleTagNumberFound(ctx context.Context, payload *sharedevents.RoundTagLookupResultPayload) ([]handlerwrapper.Result, error)
	HandleTagNumberNotFound(ctx context.Context, payload *sharedevents.RoundTagLookupResultPayload) ([]handlerwrapper.Result, error)
	HandleTagNumberLookupFailed(ctx context.Context, payload *sharedevents.RoundTagLookupFailedPayload) ([]handlerwrapper.Result, error)

	// Tag update handler
	HandleScheduledRoundTagUpdate(ctx context.Context, payload *leaderboardevents.TagUpdateForScheduledRoundsPayloadV1) ([]handlerwrapper.Result, error)

	// Round retrieval handler
	HandleGetRoundRequest(ctx context.Context, payload *roundevents.GetRoundRequestPayloadV1) ([]handlerwrapper.Result, error)

	// Round reminder handler
	HandleRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) ([]handlerwrapper.Result, error)

	// Discord message ID update handler
	HandleDiscordMessageIDUpdated(ctx context.Context, payload *roundevents.RoundScheduledPayloadV1) ([]handlerwrapper.Result, error)

	// Scorecard import handlers
	HandleScorecardUploaded(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScorecardURLRequested(ctx context.Context, payload *roundevents.ScorecardURLRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleParseScorecardRequest(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error)
	HandleUserMatchConfirmedForIngest(ctx context.Context, payload *userevents.UDiscMatchConfirmedPayload) ([]handlerwrapper.Result, error)
	HandleImportCompleted(ctx context.Context, payload *roundevents.ImportCompletedPayloadV1) ([]handlerwrapper.Result, error)
}
