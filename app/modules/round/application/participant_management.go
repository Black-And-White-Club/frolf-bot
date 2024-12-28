package roundservice

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// JoinRound handles the ParticipantResponseEvent.
func (s *RoundService) JoinRound(ctx context.Context, event *roundevents.ParticipantResponseEvent) error {
	// 1. Check if the participant has a tag number
	tagNumber, err := s.getTagNumber(ctx, event.Participant)
	if err != nil {
		return fmt.Errorf("failed to get tag number: %w", err)
	}

	// 2. Update the participant's response in the database
	participant := rounddb.Participant{
		DiscordID: event.Participant,
		Response:  rounddb.Response(event.Response),
	}
	if tagNumber != nil {
		participant.TagNumber = tagNumber
	}

	err = s.RoundDB.UpdateParticipant(ctx, event.RoundID, participant)
	if err != nil {
		return fmt.Errorf("failed to update participant response: %w", err)
	}

	// 3. Publish a ParticipantJoinedRoundEvent
	joinedEvent := &roundevents.ParticipantJoinedRoundEvent{
		RoundID:     event.RoundID,
		Participant: event.Participant,
		Response:    event.Response,
	}
	if tagNumber != nil {
		joinedEvent.TagNumber = *tagNumber // Dereference the tag number for the event
	}
	err = s.publishEvent(ctx, roundevents.ParticipantJoinedSubject, joinedEvent)
	if err != nil {
		return fmt.Errorf("failed to publish participant joined event: %w", err)
	}

	return nil
}

// UpdateScore handles the ScoreUpdatedEvent during a round.
func (s *RoundService) UpdateScore(ctx context.Context, event *roundevents.ScoreUpdatedEvent) error {
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
		err = s.FinalizeRound(ctx, &roundevents.RoundFinalizedEvent{RoundID: event.RoundID})
		if err != nil {
			return fmt.Errorf("failed to finalize round: %w", err)
		}
	} else {
		// 5. Publish a ScoreUpdatedEvent to notify the Discord bot
		err = s.publishEvent(ctx, roundevents.ScoreUpdatedSubject, &roundevents.ScoreUpdatedEvent{
			RoundID:     event.RoundID,
			Participant: event.Participant,
			Score:       event.Score,
		})
		if err != nil {
			return fmt.Errorf("failed to publish score updated event: %w", err)
		}
	}

	return nil
}

// UpdateScoreAdmin handles manual score updates after a round is finalized.
func (s *RoundService) UpdateScoreAdmin(ctx context.Context, event *roundevents.ScoreUpdatedEvent) error {
	// 1. Check the user's role (using the helper function)
	userRole, err := s.getUserRole(ctx, event.Participant)
	if err != nil {
		return fmt.Errorf("failed to get user role: %w", err)
	}

	if userRole != "Editor" && userRole != "Admin" { // Check against the string values
		return fmt.Errorf("user does not have permission to update scores")
	}

	// 2. Check if the round is finalized
	roundState, err := s.RoundDB.GetRoundState(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round state: %w", err)
	}

	if roundState != rounddb.RoundStateFinalized {
		return fmt.Errorf("cannot update score for a round that is not finalized")
	}

	// 3. Update the score in the database
	err = s.RoundDB.UpdateParticipantScore(ctx, event.RoundID, event.Participant, event.Score)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}

	// 4. Log the updated round data
	if err := s.logRoundData(ctx, event.RoundID, rounddb.ScoreUpdateTypeManual); err != nil {
		return fmt.Errorf("failed to log round data: %w", err)
	}

	// 5. Send the updated score to the Score Module
	err = s.sendRoundDataToScoreModule(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to send round data to score module: %w", err)
	}

	// 6. Publish a ScoreUpdatedEvent to notify the Discord bot
	err = s.publishEvent(ctx, roundevents.ScoreUpdatedSubject, &roundevents.ScoreUpdatedEvent{
		RoundID:     event.RoundID,
		Participant: event.Participant,
		Score:       event.Score,
	})
	if err != nil {
		return fmt.Errorf("failed to publish score updated event: %w", err)
	}

	return nil
}
