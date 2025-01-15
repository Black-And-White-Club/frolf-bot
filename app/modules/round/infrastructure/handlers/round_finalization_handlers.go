package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleFinalizeRound handles the RoundFinalizedEvent.
func (h *RoundHandlers) HandleFinalizeRound(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event roundevents.RoundFinalizedPayload // Use the correct event type
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundFinalizedEvent: %w", err)
	}

	// Call the FinalizeRound service function
	if err := h.RoundService.FinalizeRound(ctx, &event); err != nil {
		return fmt.Errorf("failed to finalize round: %w", err)
	}

	return nil
}
