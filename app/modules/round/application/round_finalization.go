package roundservice

import (
	"context"
	"fmt"
	"strconv"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
)

// FinalizeRound handles the RoundFinalizedEvent.
func (s *RoundService) FinalizeRound(ctx context.Context, event *roundevents.RoundFinalizedPayload) error {
	// 1. Update the round state to finalized
	err := s.RoundDB.UpdateRoundState(ctx, event.RoundID, rounddb.RoundStateFinalized)
	if err != nil {
		s.logger.Error("failed to update round state to finalized", "roundID", event.RoundID, "error", err)
		// Consider publishing a RoundFinalizationFailed event here
		return fmt.Errorf("failed to update round state to finalized: %w", err)
	}

	// 2. Fetch the round data ONCE
	round, err := s.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		s.logger.Error("failed to fetch round data", "roundID", event.RoundID, "error", err)
		return fmt.Errorf("failed to fetch round data: %w", err)
	}

	// 3. Log the round data in the database
	if err := s.logRoundData(ctx, round, rounddb.ScoreUpdateTypeRegular); err != nil {
		s.logger.Error("failed to log round data", "roundID", event.RoundID, "error", err)
	}

	// 4. Send round data to the Score Module
	err = s.sendRoundDataToScoreModule(ctx, round)
	if err != nil {
		s.logger.Error("failed to send round data to score module", "roundID", event.RoundID, "error", err)
		// Consider publishing a RoundDataSendFailed event here
		return fmt.Errorf("failed to send round data to score module: %w", err)
	}

	// 5. Publish a RoundFinalizedEvent
	err = s.publishEvent(ctx, roundevents.RoundFinalized, &roundevents.RoundFinalizedPayload{
		RoundID: event.RoundID,
	})
	if err != nil {
		s.logger.Error("failed to publish round finalized event", "roundID", event.RoundID, "error", err)
		return fmt.Errorf("failed to publish round finalized event: %w", err)
	}

	return nil
}

// logRoundData logs the round data in the database.
func (s *RoundService) logRoundData(ctx context.Context, round *rounddb.Round, updateType rounddb.ScoreUpdateType) error {
	return s.RoundDB.LogRound(ctx, round, updateType)
}

// sendRoundDataToScoreModule sends the round data to the Score Module.
func (s *RoundService) sendRoundDataToScoreModule(ctx context.Context, round *rounddb.Round) error {
	// Prepare the data to send to the Score Module
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

	// Publish an event to the Score Module
	err := s.publishEvent(ctx, roundevents.ProcessRoundScoresRequest, &roundevents.SendScoresPayload{
		RoundID: round.ID, // Access the ID from the round object
		Scores:  scores,
	})
	if err != nil {
		return fmt.Errorf("failed to publish send scores event: %w", err)
	}

	return nil
}
