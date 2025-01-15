package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
)

// JoinRound handles the ParticipantResponseEvent.
func (s *RoundService) JoinRound(ctx context.Context, event *roundevents.ParticipantResponsePayload) error {
	// 1. Check if the participant has a tag number
	tagNumber, err := s.getTagNumber(ctx, event.Participant)
	if err != nil {
		return fmt.Errorf("failed to get tag number: %w", err)
	}

	// 2. Update the participant's response in the database
	participant := rounddb.Participant{
		DiscordID: event.Participant,
		Response:  rounddb.ResponseAccept, // Directly use ResponseAccept
	}
	if tagNumber != nil {
		participant.TagNumber = tagNumber
	}

	err = s.RoundDB.UpdateParticipant(ctx, event.RoundID, participant)
	if err != nil {
		return fmt.Errorf("failed to update participant response: %w", err)
	}

	// 3. Publish a ParticipantJoinedRoundEvent
	joinedEvent := &roundevents.ParticipantJoinedPayload{
		RoundID:     event.RoundID,
		Participant: event.Participant,
		Response:    event.Response,
	}
	if tagNumber != nil {
		joinedEvent.TagNumber = *tagNumber // Dereference the tag number for the event
	}
	err = s.publishEvent(ctx, roundevents.ParticipantJoined, joinedEvent)
	if err != nil {
		return fmt.Errorf("failed to publish participant joined event: %w", err)
	}

	return nil
}

// UpdateScore handles the ScoreUpdatedEvent during a round.
func (s *RoundService) UpdateScore(ctx context.Context, event *roundevents.ScoreUpdatedPayload) error {
	// 1. Update the score in the database
	err := s.RoundDB.UpdateParticipantScore(ctx, event.RoundID, event.Participant, event.Score)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}

	// 2. Fetch participants with ACCEPT or TENTATIVE responses
	participants, err := s.RoundDB.GetParticipantsWithResponses(ctx, event.RoundID, rounddb.ResponseAccept, rounddb.ResponseTentative)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	// 3. Check if all scores are submitted
	allScoresSubmitted := true
	for _, p := range participants {
		if p.Score == nil {
			allScoresSubmitted = false
			break
		}
	}

	if allScoresSubmitted {
		// 4. If all scores are in, trigger FinalizeRound
		err = s.FinalizeRound(ctx, &roundevents.RoundFinalizedPayload{RoundID: event.RoundID})
		if err != nil {
			return fmt.Errorf("failed to finalize round: %w", err)
		}
	}

	// Call the common update logic
	return s.updateScoreInternal(ctx, event)
}

// UpdateScoreAdmin handles the ScoreUpdatedEvent when initiated by an admin.
func (s *RoundService) UpdateScoreAdmin(ctx context.Context, event *roundevents.ScoreUpdatedPayload) error {
	// 1. Get the role of the user who initiated the event
	userRole, err := s.getUserRole(ctx, event.Participant)
	if err != nil {
		return fmt.Errorf("failed to get user role: %w", err)
	}

	// 2. Check if the user has the Admin role
	if userRole != "Admin" {
		return fmt.Errorf("user does not have permission to update score")
	}

	// Call the common update logic
	return s.updateScoreInternal(ctx, event)
}

// updateScoreInternal handles the common score update logic.
func (s *RoundService) updateScoreInternal(ctx context.Context, event *roundevents.ScoreUpdatedPayload) error {
	// 1. Fetch the updated round data
	round, err := s.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Log the updated round data
	if err := s.logRoundData(ctx, round, rounddb.ScoreUpdateTypeManual); err != nil {
		return fmt.Errorf("failed to log round data: %w", err)
	}

	// 3. Send the updated score to the Score Module
	err = s.sendRoundDataToScoreModule(ctx, round)
	if err != nil {
		return fmt.Errorf("failed to send round data to score module: %w", err)
	}

	// 4. Publish a ScoreUpdatedEvent to notify the Discord bot
	err = s.publishEvent(ctx, roundevents.ScoreUpdated, &roundevents.ScoreUpdatedPayload{
		RoundID:     event.RoundID,
		Participant: event.Participant,
		Score:       event.Score,
	})
	if err != nil {
		return fmt.Errorf("failed to publish score updated event: %w", err)
	}

	return nil
}
