package roundservice

import (
	"context"
	"fmt"
	"strconv"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// FinalizeRound handles the round finalization process.
func (s *RoundService) FinalizeRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.AllScoresSubmittedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal AllScoresSubmittedPayload: %w", err)
	}

	// 1. Update the round state to finalized
	if err := s.RoundDB.UpdateRoundState(ctx, eventPayload.RoundID, roundtypes.RoundStateFinalized); err != nil {
		return s.publishRoundFinalizationError(msg, eventPayload, err)
	}

	// 2. Publish a "round.finalized" event
	if err := s.publishEvent(msg, roundevents.RoundFinalized, roundevents.RoundFinalizedPayload(eventPayload)); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.finalized event", map[string]interface{}{})
		return fmt.Errorf("failed to publish round.finalized event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round finalized", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// NotifyScoreModule fetches the finalized round data and publishes an event for the Score Module.
func (s *RoundService) NotifyScoreModule(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundFinalizedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundFinalizedPayload: %w", err)
	}

	// 1. Fetch the finalized round data
	round, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return s.publishScoreModuleNotificationError(msg, eventPayload, err) // Specific error event
	}

	// 2. Prepare the data for the Score Module
	scores := make([]roundevents.ParticipantScore, 0)
	for _, p := range round.Participants {
		tagNumber := "0" // Default if no tag
		if p.TagNumber != 0 {
			tagNumber = strconv.Itoa(p.TagNumber)
		}

		// Use 0 for nil scores, otherwise convert to float64
		score := float64(0)
		if p.Score != nil {
			score = float64(*p.Score)
		}

		scores = append(scores, roundevents.ParticipantScore{
			DiscordID: p.DiscordID,
			TagNumber: tagNumber,
			Score:     score,
		})
	}

	// 3. Publish an event to the Score Module
	if err := s.publishEvent(msg, roundevents.ProcessRoundScoresRequest, roundevents.ProcessRoundScoresRequestPayload{
		RoundID: round.ID,
		Scores:  scores,
	}); err != nil {
		return s.publishScoreModuleNotificationError(msg, eventPayload, err) // Specific error event
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Notified Score Module about finalized round", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// publishRoundFinalizationError publishes a round.finalization.error event.
func (s *RoundService) publishRoundFinalizationError(msg *message.Message, input roundevents.AllScoresSubmittedPayload, err error) error {
	payload := roundevents.RoundFinalizationErrorPayload{
		CorrelationID: middleware.MessageCorrelationID(msg),
		RoundID:       input.RoundID,
		Error:         err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundFinalizationError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.finalization.error event", map[string]interface{}{
			"original_error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.finalization.error event: %w, original error: %w", pubErr, err)
	}

	return err
}

// publishScoreModuleNotificationError publishes a score.module.notification.error event.
func (s *RoundService) publishScoreModuleNotificationError(msg *message.Message, input roundevents.RoundFinalizedPayload, err error) error {
	payload := roundevents.ScoreModuleNotificationErrorPayload{
		CorrelationID: middleware.MessageCorrelationID(msg),
		RoundID:       input.RoundID,
		Error:         err.Error(),
	}
	if pubErr := s.publishEvent(msg, roundevents.ScoreModuleNotificationError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish score.module.notification.error event", map[string]interface{}{
			"original_error": err.Error(),
		})
		return fmt.Errorf("failed to publish score.module.notification.error event: %w, original error: %w", pubErr, err)
	}
	return err

}
