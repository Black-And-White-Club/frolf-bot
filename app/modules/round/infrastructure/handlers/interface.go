package roundhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type Handlers interface {
	// Create Round
	HandleRoundCreateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundEntityCreated(msg *message.Message) ([]*message.Message, error)
	HandleRoundStored(msg *message.Message) ([]*message.Message, error)
	HandleRoundScheduled(msg *message.Message) ([]*message.Message, error)
	HandleUpdateEventMessageID(msg *message.Message) ([]*message.Message, error)

	// Update Round
	HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundFetched(msg *message.Message) ([]*message.Message, error)
	HandleRoundEntityUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundScheduleUpdate(msg *message.Message) ([]*message.Message, error)

	// Delete Round
	HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundToDeleteFetched(msg *message.Message) ([]*message.Message, error)
	HandleRoundUserRoleCheckResult(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteAuthorized(msg *message.Message) ([]*message.Message, error)

	// Start Round
	HandleRoundStarted(msg *message.Message) ([]*message.Message, error)
	HandleRoundReminder(msg *message.Message) ([]*message.Message, error)
	HandleScheduleRoundEvents(msg *message.Message) ([]*message.Message, error)

	// Join Round
	HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundTagNumberFound(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantDeclined(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoinValidationRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantRemovalRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoinValidated(msg *message.Message) ([]*message.Message, error)

	HandleRoundTagNumberNotFound(msg *message.Message) ([]*message.Message, error)

	// Score Round
	HandleRoundScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundScoreUpdateValidated(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundAllScoresSubmitted(msg *message.Message) ([]*message.Message, error)
	HandleRoundFinalized(msg *message.Message) ([]*message.Message, error)

	// Tag Retrieval
	HandleRoundTagNumberRequest(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardGetTagNumberResponse(msg *message.Message) ([]*message.Message, error)
}
