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

// DeleteRoundHandler handles the DeleteRound command.
type DeleteRoundHandler struct {
	roundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
}

// NewDeleteRoundHandler creates a new DeleteRoundHandler.
func NewDeleteRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher) *DeleteRoundHandler {
	return &DeleteRoundHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
	}
}

// Handle processes the DeleteRound command.
func (h *DeleteRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd DeleteRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal DeleteRoundRequest: %w", err)
	}

	err := h.roundDB.DeleteRound(ctx, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	event := RoundDeletedEvent{
		RoundID: cmd.RoundID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundDeletedEvent: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish RoundDeletedEvent: %w", err)
	}

	return nil
}
