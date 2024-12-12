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

// EditRoundHandler handles the EditRound command.
type EditRoundHandler struct {
	roundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
}

// NewEditRoundHandler creates a new EditRoundHandler.
func NewEditRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher) *EditRoundHandler {
	return &EditRoundHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
	}
}

// Handle processes the EditRound command.
func (h *EditRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd EditRoundRequest // Defined in requests.go
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal EditRoundRequest: %w", err)
	}

	err := h.roundDB.UpdateRound(ctx, cmd.RoundID, cmd.Input)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	event := RoundEditedEvent{ // Defined in events.go
		RoundID: cmd.RoundID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundEditedEvent: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil { // Topic() defined in topics.go
		return fmt.Errorf("failed to publish RoundEditedEvent: %w", err)
	}

	return nil
}
