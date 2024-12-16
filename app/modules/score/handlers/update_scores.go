package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	scorecommands "github.com/Black-And-White-Club/tcr-bot/app/modules/score/commands"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// UpdateScoresHandler handles updating scores for a round.
type UpdateScoresHandler struct {
	eventBus watermillutil.PubSuber
	scoredb  scoredb.ScoreDB
}

// NewUpdateScoresHandler creates a new UpdateScoresHandler instance.
func NewUpdateScoresHandler(eventBus watermillutil.PubSuber, scoredb scoredb.ScoreDB) *UpdateScoresHandler {
	return &UpdateScoresHandler{
		eventBus: eventBus,
		scoredb:  scoredb,
	}
}

// Handle updates the scores and triggers the score processing pipeline.
func (h *UpdateScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd scorecommands.UpdateScoresCommand
	marshaler := watermillutil.Marshaler
	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal UpdateScoresCommand")
	}

	// 1. Update or add scores in the database
	if err := h.updateScores(ctx, cmd.RoundID, cmd.Scores); err != nil {
		return fmt.Errorf("failed to update scores in the database: %w", err)
	}

	// 2. Fetch all scores for the round
	scores, err := h.scoredb.GetScoresForRound(ctx, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to fetch scores for round: %w", err)
	}

	// 3. Publish ScoresReceivedEvent to trigger the processing pipeline
	event := ScoresReceivedEvent{
		RoundID: cmd.RoundID,
		Scores:  scores,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ScoresReceivedEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicReceiveScores, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ScoresReceivedEvent: %w", err)
	}

	return nil
}

// updateScores updates or adds scores in the database.
func (h *UpdateScoresHandler) updateScores(ctx context.Context, roundID string, scores []scoredb.Score) error {
	for _, score := range scores {
		// Check if the score already exists
		existingScore, err := h.scoredb.GetScore(ctx, score.DiscordID, roundID) // Use h.scoredb instead of h.scoredb
		if err != nil {
			return fmt.Errorf("failed to check for existing score: %w", err)
		}

		if existingScore != nil {
			// Update the existing score
			existingScore.Score = score.Score
			existingScore.TagNumber = score.TagNumber
			if err := h.scoredb.UpdateScore(ctx, existingScore); err != nil {
				return fmt.Errorf("failed to update score: %w", err)
			}
		} else {
			// Insert a new score
			newScore := scoredb.Score{
				DiscordID: score.DiscordID,
				RoundID:   roundID,
				Score:     score.Score,
				TagNumber: score.TagNumber,
			}
			if err := h.scoredb.InsertScore(ctx, &newScore); err != nil { // Use h.scoredb instead of h.scoredb
				return fmt.Errorf("failed to insert score: %w", err)
			}
		}
	}

	return nil
}
