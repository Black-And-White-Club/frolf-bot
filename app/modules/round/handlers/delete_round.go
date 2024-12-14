package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/service"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// DeleteRoundHandler handles the DeleteRound command.
type DeleteRoundHandler struct {
	roundDB      rounddb.RoundDB
	messageBus   watermillutil.Publisher
	roundService roundservice.Service
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
	var cmd roundcommands.DeleteRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal DeleteRoundRequest: %w", err)
	}

	// 1. Check if the round is upcoming
	isUpcoming, err := h.roundService.IsRoundUpcoming(ctx, cmd.RoundID)
	if err != nil {
		return err // Or handle the error more specifically
	}
	if !isUpcoming {
		return fmt.Errorf("cannot delete round that is not upcoming")
	}

	// 2. Update the round's state to DELETED
	err = h.roundDB.UpdateRoundState(ctx, cmd.RoundID, rounddb.RoundStateDeleted) // Use db.RoundState
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// 3. Publish RoundDeletedEvent
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
