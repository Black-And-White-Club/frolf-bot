package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCreateRound handles the RoundCreatedEvent.
func (h *RoundHandlers) HandleCreateRound(msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundCreatedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundCreatedEvent: %w", err)
	}

	// Call the CreateRound service function
	// Assuming you have a way to extract the CreateRoundParams from the message
	// or have a different event that carries the CreateRoundParams
	var params rounddto.CreateRoundParams // Replace with actual params extraction
	if err := h.RoundService.CreateRound(context.Background(), &event, params); err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	return nil
}

// HandleUpdateRound handles the RoundUpdatedEvent.
func (h *RoundHandlers) HandleUpdateRound(msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundUpdatedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdatedEvent: %w", err)
	}

	if err := h.RoundService.UpdateRound(context.Background(), &event); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	return nil
}

// HandleDeleteRound handles the RoundDeletedEvent.
func (h *RoundHandlers) HandleDeleteRound(msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundDeletedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeletedEvent: %w", err)
	}

	if err := h.RoundService.DeleteRound(context.Background(), &event); err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	return nil
}
