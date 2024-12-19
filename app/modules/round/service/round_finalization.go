package roundservice

import (
	"context"
	"fmt"
	"strconv"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// FinalizeRound handles the RoundFinalizedEvent.
func (s *RoundService) FinalizeRound(ctx context.Context, event *roundevents.RoundFinalizedEvent) error {
	// 1. Update the round state to finalized
	err := s.RoundDB.UpdateRoundState(ctx, event.RoundID, rounddb.RoundStateFinalized)
	if err != nil {
		return fmt.Errorf("failed to update round state to finalized: %w", err)
	}

	// 2. Log the round data in the database
	if err := s.logRoundData(ctx, event.RoundID, rounddb.ScoreUpdateTypeRegular); err != nil {
		return fmt.Errorf("failed to log round data: %w", err)
	}

	// 3. Send round data to the Score Module
	err = s.sendRoundDataToScoreModule(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to send round data to score module: %w", err)
	}

	// 4. Publish a RoundFinalizedEvent
	err = s.publishEvent(ctx, "round.finalized", &roundevents.RoundFinalizedEvent{
		RoundID: event.RoundID,
		// ... (Include any other necessary data in the event)
	})
	if err != nil {
		return fmt.Errorf("failed to publish round finalized event: %w", err)
	}

	return nil
}

// logRoundData logs the round data in the database.
func (s *RoundService) logRoundData(ctx context.Context, roundID string, updateType rounddb.ScoreUpdateType) error {
	round, err := s.RoundDB.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	return s.RoundDB.LogRound(ctx, round, updateType)
}

// sendRoundDataToScoreModule sends the round data to the Score Module.
func (s *RoundService) sendRoundDataToScoreModule(ctx context.Context, roundID string) error {
	// 1. Fetch the round data from the database
	round, err := s.RoundDB.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Prepare the data to send to the Score Module
	scores := make([]roundevents.ParticipantScore, 0)
	for _, p := range round.Participants {
		// Only include participants with scores
		if p.Score != nil {
			tagNumber := "0" // Default tag number if none is assigned
			if p.TagNumber != nil {
				tagNumber = strconv.Itoa(*p.TagNumber)
			}
			scores = append(scores, roundevents.ParticipantScore{
				DiscordID: p.DiscordID,
				TagNumber: tagNumber,
				Score:     *p.Score,
			})
		}
	}

	// 3. Publish an event to the Score Module
	err = s.publishEvent(ctx, "score.process_round_scores", &roundevents.SendScoresEvent{
		RoundID: roundID,
		Scores:  scores,
	})
	if err != nil {
		return fmt.Errorf("failed to publish send scores event: %w", err)
	}

	return nil
}
