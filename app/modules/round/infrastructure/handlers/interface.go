package roundhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type Handlers interface {
	HandleRoundStarted(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateValidated(msg *message.Message) ([]*message.Message, error)
	HandleAllScoresSubmitted(msg *message.Message) ([]*message.Message, error)
	HandleRoundFinalized(msg *message.Message) ([]*message.Message, error)
	HandleGetRoundRequest(msg *message.Message) ([]*message.Message, error)
	HandleParticipantJoinRequest(msg *message.Message) ([]*message.Message, error)
	HandleParticipantJoinValidationRequest(msg *message.Message) ([]*message.Message, error)
	HandleParticipantRemovalRequest(msg *message.Message) ([]*message.Message, error)
	HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteAuthorized(msg *message.Message) ([]*message.Message, error)
	HandleCreateRoundRequest(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundReminder(msg *message.Message) ([]*message.Message, error)
	HandleRoundEntityCreated(msg *message.Message) ([]*message.Message, error)
	HandleParticipantStatusUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleParticipantDeclined(msg *message.Message) ([]*message.Message, error)
	HandleRoundEventMessageIDUpdate(msg *message.Message) ([]*message.Message, error)
	HandleDiscordMessageIDUpdated(msg *message.Message) ([]*message.Message, error)
	HandleTagNumberFound(msg *message.Message) ([]*message.Message, error)
	HandleTagNumberNotFound(msg *message.Message) ([]*message.Message, error)
	HandleTagNumberLookupFailed(msg *message.Message) ([]*message.Message, error)
	HandleScheduledRoundTagUpdate(msg *message.Message) ([]*message.Message, error)
	HandleRoundScheduleUpdate(msg *message.Message) ([]*message.Message, error)
	// Scorecard import handlers
	HandleScorecardUploaded(msg *message.Message) ([]*message.Message, error)
	HandleScorecardURLRequested(msg *message.Message) ([]*message.Message, error)
	HandleParseScorecardRequest(msg *message.Message) ([]*message.Message, error)
	HandleScorecardParsed(msg *message.Message) ([]*message.Message, error)
}
