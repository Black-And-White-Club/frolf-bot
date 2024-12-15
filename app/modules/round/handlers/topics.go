package roundhandlers

// Topic returns the topic for the FinalizeRoundEvent.
func (e FinalizeRoundEvent) Topic() string {
	return "round.finalize"
}

func (e RoundDeletedEvent) Topic() string {
	return "round.deleted"
}

// Topic returns the topic for the RoundEditedEvent.
func (e RoundEditedEvent) Topic() string {
	return "round.edited"
}

// Topic returns the topic for the RoundUpdatedEvent.
func (e RoundStateUpdatedEvent) Topic() string {
	return "round.state.updated"
}

// Topic returns the topic for the RoundCreated event.
func (e RoundCreatedEvent) Topic() string {
	return "round.created"
}

// Topic returns the topic for the RoundReminderEvent.
func (e RoundReminderEvent) Topic() string {
	return "round.reminder"
}

// Topic returns the topic for the ParticipantJoinedRoundEvent.
func (e ParticipantJoinedRoundEvent) Topic() string {
	return "participant.joined.round"
}

// Topic returns the topic for the SendScoresEvent.
func (e SendScoresEvent) Topic() string {
	return "round.scores.send"
}
