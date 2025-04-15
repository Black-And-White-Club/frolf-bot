package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// GetRoundsAndParticipantsToUpdate returns the rounds and participants that need to be updated.
func (s *RoundService) getRoundsAndParticipantsToUpdate(ctx context.Context, changedTags map[sharedtypes.DiscordID]*sharedtypes.TagNumber) ([]roundtypes.RoundUpdate, error) {
	// Get the upcoming rounds
	rounds, err := s.RoundDB.GetUpcomingRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}

	// Create a slice to store the rounds and participants that need to be updated
	updates := make([]roundtypes.RoundUpdate, 0)

	// Iterate over the rounds and find the participants that need to be updated
	for _, round := range rounds {
		update := roundtypes.RoundUpdate{
			RoundID:        round.ID,
			EventMessageID: round.ID,
			Participants:   make([]roundtypes.Participant, 0),
		}
		for _, participant := range round.Participants {
			if newTag, exists := changedTags[participant.UserID]; exists {
				participant.TagNumber = newTag
				update.Participants = append(update.Participants, participant)
			}
		}
		if len(update.Participants) > 0 {
			updates = append(updates, update)
		}
	}

	s.logger.InfoContext(ctx, "Rounds and participants to update retrieved",
		attr.Int("num_updates", len(updates)),
	)

	return updates, nil
}

// UpdateScheduledRoundsWithNewTags updates the scheduled rounds with the new participant tags.
func (s *RoundService) UpdateScheduledRoundsWithNewTags(ctx context.Context, payload roundevents.ScheduledRoundTagUpdatePayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "UpdateScheduledRoundsWithNewTags", func() (RoundOperationResult, error) {
		// Get the rounds and participants that need to be updated
		updates, err := s.getRoundsAndParticipantsToUpdate(ctx, payload.ChangedTags)
		if err != nil {
			return RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					Error: err.Error(),
				},
			}, fmt.Errorf("failed to get rounds and participants: %w", err)
		}

		s.logger.InfoContext(ctx, "Round updates created",
			attr.Int("num_updates", len(updates)),
		)

		// Update the rounds and participants in the DB
		if err := s.RoundDB.UpdateRoundsAndParticipants(ctx, updates); err != nil {
			return RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					Error: err.Error(),
				},
			}, fmt.Errorf("failed to update rounds: %w", err)
		}

		s.logger.InfoContext(ctx, "Rounds updated successfully",
			attr.Int("num_updates", len(updates)),
		)

		// Create the Discord round update payload
		discordUpdatePayload := roundevents.DiscordRoundUpdatePayload{
			Participants:    make([]roundtypes.Participant, 0),
			RoundIDs:        make([]sharedtypes.RoundID, 0),
			EventMessageIDs: make([]sharedtypes.RoundID, 0),
		}

		// Iterate over the updates and add the updated participants to the payload
		for _, update := range updates {
			discordUpdatePayload.Participants = append(discordUpdatePayload.Participants, update.Participants...)
			discordUpdatePayload.RoundIDs = append(discordUpdatePayload.RoundIDs, update.RoundID)
			discordUpdatePayload.EventMessageIDs = append(discordUpdatePayload.EventMessageIDs, update.EventMessageID)
		}

		return RoundOperationResult{Success: discordUpdatePayload}, nil
	})

	return result, err
}
