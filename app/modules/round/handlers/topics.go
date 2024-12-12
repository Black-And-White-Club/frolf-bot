package roundhandlers

func (e RoundDeletedEvent) Topic() string {
	return "round.deleted"
}

// Topic returns the topic for the RoundEditedEvent.
func (e RoundEditedEvent) Topic() string {
	return "round.edited"
}
