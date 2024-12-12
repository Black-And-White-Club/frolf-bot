package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ... (other structs)

// RoundScoresProcessed represents the event of scores being processed for a round.
type RoundScoresProcessed struct {
	RoundID      int64                `json:"round_id"`
	Participants []models.Participant `json:"participants"`
	Scores       map[string]int       `json:"scores"`
}

// Topic returns the topic for the RoundScoresProcessed event.
func (e RoundScoresProcessed) Topic() string {
	return "round.scores.processed"
}

// ... (FinalizeAndProcessScoresHandler struct and constructor)

func (h *FinalizeAndProcessScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	// ... (unmarshal command, get round, check if already finalized)

	// Process scores using the new handler
	processScoresCmd := ProcessScoresRequest{
		RoundID:      cmd.RoundID,
		Participants: round.Participants,
		Scores:       round.Scores,
	}
	if err := h.messageBus.Publish(processScoresCmd.CommandName(), message.NewMessage(watermill.NewUUID(), processScoresCmd)); err != nil {
		return fmt.Errorf("failed to publish ProcessScoresRequest: %w", err)
	}

	if round.State != models.RoundStateFinalized {
		// Publish command to update the round state ONLY if not already finalized
		updateStateCmd := UpdateRoundStateRequest{
			RoundID: cmd.RoundID,
			State:   models.RoundStateFinalized,
		}
		if err := h.messageBus.Publish(updateStateCmd.CommandName(), message.NewMessage(watermill.NewUUID(), updateStateCmd)); err != nil {
			return fmt.Errorf("failed to publish UpdateRoundStateRequest: %w", err)
		}
	}

	// Publish RoundScoresProcessed event regardless of the round state
	event := RoundScoresProcessed{
		RoundID:      cmd.RoundID,
		Participants: round.Participants, // Include the participant data
		Scores:       round.Scores,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundScoresProcessed event: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish RoundScoresProcessed event: %w", err)
	}

	return nil
}
