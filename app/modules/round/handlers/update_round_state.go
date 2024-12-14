package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UpdateRoundStateHandler handles the UpdateRoundState command.
type UpdateRoundStateHandler struct {
	roundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
	logger     watermill.LoggerAdapter
}

// NewUpdateRoundStateHandler creates a new UpdateRoundStateHandler.
func NewUpdateRoundStateHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher, logger watermill.LoggerAdapter) *UpdateRoundStateHandler {
	return &UpdateRoundStateHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
		logger:     logger,
	}
}

// Handle processes the UpdateRoundState command.
func (h *UpdateRoundStateHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd roundcommands.UpdateRoundStateRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		h.logger.Error("Failed to unmarshal UpdateRoundStateRequest", err, watermill.LogFields{"payload": string(msg.Payload)})
		return fmt.Errorf("failed to unmarshal UpdateRoundStateRequest: %w", err)
	}

	h.logger.Info("Handling UpdateRoundStateRequest", watermill.LogFields{
		"round_id": cmd.RoundID,
		"state":    cmd.State,
	})

	err := h.roundDB.UpdateRoundState(ctx, cmd.RoundID, cmd.State)
	if err != nil {
		h.logger.Error("Failed to update round state in DB", err, watermill.LogFields{"round_id": cmd.RoundID, "state": cmd.State})
		return fmt.Errorf("failed to update round state: %w", err)
	}

	event := RoundStateUpdatedEvent{
		RoundID: cmd.RoundID,
		State:   cmd.State,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal RoundStateUpdatedEvent", err, watermill.LogFields{"round_id": cmd.RoundID, "state": cmd.State})
		return fmt.Errorf("failed to marshal RoundStateUpdatedEvent: %w", err)
	}

	h.logger.Info("Publishing RoundStateUpdatedEvent", watermill.LogFields{
		"round_id": cmd.RoundID,
		"state":    cmd.State,
	})

	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		h.logger.Error("Failed to publish RoundStateUpdatedEvent", err, watermill.LogFields{"round_id": cmd.RoundID, "state": cmd.State})
		return fmt.Errorf("failed to publish RoundStateUpdatedEvent: %w", err)
	}

	return nil
}
