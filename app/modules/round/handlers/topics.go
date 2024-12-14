package roundhandlers

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

// Topic returns the topic for the ParticipantJoinedRoundEvent.
func (e ParticipantJoinedRoundEvent) Topic() string {
	return "participant.joined.round"
}

// Topic returns the topic for the GetTagNumberRequest event.
func (e GetTagNumberRequest) Topic() string {
	return "get.tag.number.request"
}
