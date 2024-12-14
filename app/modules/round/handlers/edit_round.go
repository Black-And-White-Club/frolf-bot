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

// EditRoundHandler handles the EditRound command.
type EditRoundHandler struct {
	roundDB      rounddb.RoundDB
	messageBus   watermillutil.Publisher
	roundService roundservice.Service
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
	var cmd roundcommands.EditRoundRequest // Defined in requests.go
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal EditRoundRequest: %w", err)
	}

	// Construct the updates map
	updates := map[string]interface{}{
		"title":      cmd.APIInput.Title,
		"location":   cmd.APIInput.Location,
		"event_type": cmd.APIInput.EventType,
		"date":       cmd.APIInput.Date,
		"time":       cmd.APIInput.Time,
	}

	err := h.roundDB.UpdateRound(ctx, cmd.RoundID, updates)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	// 1. Check if the round is upcoming
	isUpcoming, err := h.roundService.IsRoundUpcoming(ctx, cmd.RoundID)
	if err != nil {
		return err // Or handle the error more specifically
	}
	if !isUpcoming {
		return fmt.Errorf("cannot edit round that is not upcoming")
	}

	event := RoundEditedEvent{ // Defined in events.go
		RoundID: cmd.RoundID,
		Updates: updates,
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
