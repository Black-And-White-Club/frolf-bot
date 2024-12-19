package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleParticipantResponse handles the ParticipantResponseEvent.
func (h *RoundHandlers) HandleParticipantResponse(msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.ParticipantResponseEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantResponseEvent: %w", err)
	}

	if err := h.RoundService.JoinRound(context.Background(), &event); err != nil {
		return fmt.Errorf("failed to process participant response: %w", err)
	}

	return nil
}

// HandleScoreUpdated handles the ScoreUpdatedEvent.
func (h *RoundHandlers) HandleScoreUpdated(msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.ScoreUpdatedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal ScoreUpdatedEvent: %w", err)
	}

	// Check the update type to determine which service function to call
	switch event.UpdateType { // Use event.UpdateType
	case rounddb.ScoreUpdateTypeRegular:
		if err := h.RoundService.UpdateScore(context.Background(), &event); err != nil {
			return fmt.Errorf("failed to process score update: %w", err)
		}
	case rounddb.ScoreUpdateTypeManual:
		if err := h.RoundService.UpdateScoreAdmin(context.Background(), &event); err != nil {
			return fmt.Errorf("failed to process admin score update: %w", err)
		}
	default:
		return fmt.Errorf("invalid score update type: %d", event.UpdateType) // Use event.UpdateType
	}

	return nil
}
