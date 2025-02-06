package roundhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type Handlers interface {
	// Create Round
	HandleRoundCreateRequest(msg *message.Message) error
	HandleRoundValidated(msg *message.Message) error
	HandleRoundDateTimeParsed(msg *message.Message) error
	HandleRoundStored(msg *message.Message) error
	HandleRoundScheduled(msg *message.Message) error

	// Update Round
	HandleRoundUpdateRequest(msg *message.Message) error
	HandleRoundUpdateValidated(msg *message.Message) error
	HandleRoundFetched(msg *message.Message) error
	HandleRoundEntityUpdated(msg *message.Message) error
	HandleRoundScheduleUpdate(msg *message.Message) error

	// Delete Round
	HandleRoundDeleteRequest(msg *message.Message) error
	HandleRoundDeleteValidated(msg *message.Message) error
	HandleRoundToDeleteFetched(msg *message.Message) error
	HandleRoundUserRoleCheckResult(msg *message.Message) error
	HandleRoundDeleteAuthorized(msg *message.Message) error

	// Start Round
	HandleRoundStarted(msg *message.Message) error
	HandleRoundReminder(msg *message.Message) error
	HandleScheduleRoundEvents(msg *message.Message) error

	// Join Round
	HandleRoundParticipantJoinRequest(msg *message.Message) error
	HandleRoundParticipantJoinValidated(msg *message.Message) error
	HandleRoundTagNumberFound(msg *message.Message) error
	HandleRoundTagNumberNotFound(msg *message.Message) error

	// Score Round
	HandleRoundScoreUpdateRequest(msg *message.Message) error
	HandleRoundScoreUpdateValidated(msg *message.Message) error
	HandleRoundParticipantScoreUpdated(msg *message.Message) error
	HandleRoundAllScoresSubmitted(msg *message.Message) error
	HandleRoundFinalized(msg *message.Message) error

	// Tag Retrieval
	HandleRoundTagNumberRequest(msg *message.Message) error
	HandleLeaderboardGetTagNumberResponse(msg *message.Message) error
}
