package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCreateRound handles the RoundCreatedEvent.
func (h *RoundHandlers) HandleCreateRound(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundCreateRequestPayload // Use the correct event type
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundCreatedEvent: %w", err)
	}

	// Call the CreateRound service function
	if err := h.RoundService.CreateRound(ctx, event); err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	return nil
}

// HandleUpdateRound handles the RoundUpdatedEvent.
func (h *RoundHandlers) HandleUpdateRound(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundUpdatedPayload // Use the correct event type
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdatedEvent: %w", err)
	}

	if err := h.RoundService.UpdateRound(ctx, &event); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	return nil
}

// HandleDeleteRound handles the RoundDeletedEvent.
func (h *RoundHandlers) HandleDeleteRound(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundDeletedPayload // Use the correct event type
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeletedEvent: %w", err)
	}

	if err := h.RoundService.DeleteRound(ctx, &event); err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	return nil
}

// HandleStartRound handles the RoundStartedEvent.
func (h *RoundHandlers) HandleStartRound(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundStartedPayload // Use the correct event type
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundStartedEvent: %w", err)
	}

	if err := h.RoundService.StartRound(ctx, &event); err != nil {
		return fmt.Errorf("failed to start round: %w", err)
	}

	return nil
}
