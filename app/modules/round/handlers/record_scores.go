package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RecordScoresHandler handles the RecordScores command.
type RecordScoresHandler struct {
	RoundDB  rounddb.RoundDB
	PubSuber watermillutil.PubSuber
}

// NewRecordScoresHandler creates a new RecordScoresHandler.
func NewRecordScoresHandler(roundDB rounddb.RoundDB, pubsuber watermillutil.PubSuber) *RecordScoresHandler {
	return &RecordScoresHandler{
		RoundDB:  roundDB,
		PubSuber: pubsuber,
	}
}

// Handle processes the RecordScores command.
func (h *RecordScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event RoundScoresProcessedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundScoresProcessed: %w", err)
	}

	// Fetch the round from the database
	round, err := h.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round is finalized
	if round.State != rounddb.RoundStateFinalized {
		return fmt.Errorf("cannot record scores for a non-finalized round")
	}

	// 1. Record the scores in the database (move from PendingScores to Scores)
	round.Scores = make(map[string]int)
	for _, score := range round.PendingScores {
		round.Scores[score.ParticipantID] = score.Score
	}
	round.PendingScores = nil // Clear the PendingScores
	if err := h.RoundDB.RecordScores(ctx, round.ID, round.Scores); err != nil {
		return fmt.Errorf("failed to update scores: %w", err)
	}

	// 2. Publish an event to trigger the SendScoresHandler
	var sendScores []ParticipantScore
	for _, participant := range round.Participants {
		scoreKey := fmt.Sprintf("%d", *participant.TagNumber)
		score, ok := round.Scores[scoreKey]
		if !ok {
			return fmt.Errorf("score not found for participant %s", participant.DiscordID)
		}
		sendScores = append(sendScores, ParticipantScore{
			DiscordID: participant.DiscordID,
			TagNumber: *participant.TagNumber,
			Score:     score,
		})
	}

	sendScoresEvent := &SendScoresEvent{
		RoundID: round.ID,
		Scores:  sendScores,
	}
	sendScoresPayload, err := json.Marshal(sendScoresEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal SendScoresEvent: %w", err)
	}
	if err := h.PubSuber.Publish(sendScoresEvent.Topic(), message.NewMessage(watermill.NewUUID(), sendScoresPayload)); err != nil {
		return fmt.Errorf("failed to publish SendScoresEvent: %w", err)
	}

	return nil
}
