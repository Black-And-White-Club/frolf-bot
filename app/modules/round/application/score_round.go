package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/shared/logging"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for Score Update Flow --

// ValidateScoreUpdateRequest validates the score update request.
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ScoreUpdateRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishScoreUpdateError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	var errs []string
	if eventPayload.RoundID == "" {
		errs = append(errs, "round ID cannot be empty")
	}
	if eventPayload.Participant == "" {
		errs = append(errs, "participant Discord ID cannot be empty")
	}
	if eventPayload.Score == nil {
		errs = append(errs, "score cannot be empty")
	}
	// Add more validation rules as needed...

	if len(errs) > 0 {
		err := fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Score update request validation failed", map[string]interface{}{
			"errors": errs,
		})
		return s.publishScoreUpdateError(msg, eventPayload, err) // Publishes round.score.update.error
	}

	// If validation passes, publish a "round.score.update.validated" event
	if err := s.publishEvent(msg, roundevents.RoundScoreUpdateValidated, roundevents.ScoreUpdateValidatedPayload{
		ScoreUpdateRequestPayload: eventPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.score.update.validated event", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.score.update.validated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Score update request validated", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// UpdateParticipantScore updates the participant's score in the database.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ScoreUpdateValidatedPayload](msg, s.logger)
	if err != nil {
		return s.publishScoreUpdateError(msg, eventPayload.ScoreUpdateRequestPayload, fmt.Errorf("invalid payload: %w", err))
	}

	// Access fields through eventPayload.ScoreUpdateRequestPayload
	err = s.RoundDB.UpdateParticipantScore(ctx, eventPayload.ScoreUpdateRequestPayload.RoundID, eventPayload.ScoreUpdateRequestPayload.Participant, *eventPayload.ScoreUpdateRequestPayload.Score)
	if err != nil {
		return s.publishScoreUpdateError(msg, eventPayload.ScoreUpdateRequestPayload, err) // Error event
	}

	// Publish a "round.participant.score.updated" event
	if err := s.publishEvent(msg, roundevents.RoundParticipantScoreUpdated, roundevents.ParticipantScoreUpdatedPayload{
		RoundID:     eventPayload.ScoreUpdateRequestPayload.RoundID,
		Participant: eventPayload.ScoreUpdateRequestPayload.Participant,
		Score:       *eventPayload.ScoreUpdateRequestPayload.Score,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.score.updated event", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.participant.score.updated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant score updated in database", map[string]interface{}{
		"round_id":       eventPayload.ScoreUpdateRequestPayload.RoundID,
		"participant_id": eventPayload.ScoreUpdateRequestPayload.Participant,
		"score":          *eventPayload.ScoreUpdateRequestPayload.Score,
	})
	return nil
}

// CheckAllScoresSubmitted checks if all participants in the round have submitted scores.
func (s *RoundService) CheckAllScoresSubmitted(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantScoreUpdatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantScoreUpdatedPayload: %w", err)
	}

	// Get all participants for the round
	participants, err := s.RoundDB.GetParticipants(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get participants for round: %w, %w", err, s.publishScoreUpdateError(msg, roundevents.ScoreUpdateRequestPayload{}, err))
	}

	// Check if all participants have a score
	allScoresSubmitted := true
	for _, p := range participants {
		if p.Score == nil {
			allScoresSubmitted = false
			break
		}
	}

	if allScoresSubmitted {
		// Publish a "round.all.scores.submitted" event
		if err := s.publishEvent(msg, roundevents.RoundAllScoresSubmitted, roundevents.AllScoresSubmittedPayload{
			RoundID: eventPayload.RoundID,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.all.scores.submitted event", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to publish round.all.scores.submitted event: %w", err)
		}
		logging.LogInfoWithMetadata(ctx, s.logger, msg, "All scores submitted for round", map[string]interface{}{"round_id": eventPayload.RoundID})
	} else {
		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Not all scores have been submitted for round", map[string]interface{}{"round_id": eventPayload.RoundID})
	}

	return nil
}

// publishScoreUpdateError publishes a round.score.update.error event.
func (s *RoundService) publishScoreUpdateError(msg *message.Message, input roundevents.ScoreUpdateRequestPayload, err error) error {
	payload := roundevents.RoundScoreUpdateErrorPayload{
		CorrelationID:      middleware.MessageCorrelationID(msg),
		ScoreUpdateRequest: &input,
		Error:              err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundScoreUpdateError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.score.update.error event", map[string]interface{}{
			"original_error": err.Error(),
			"publish_error":  pubErr.Error(),
		})
		return fmt.Errorf("failed to publish round.score.update.error event: %w, original error: %w", pubErr, err)
	}

	return err
}
