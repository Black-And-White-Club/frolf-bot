package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/models"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UpdateRoundStateRequest represents the request to update the state of a round.
type UpdateRoundStateRequest struct {
	RoundID int64             `json:"round_id"`
	State   models.RoundState `json:"state"`
}

// CommandName returns the command name for UpdateRoundStateRequest
func (cmd UpdateRoundStateRequest) CommandName() string {
	return "update_round_state"
}

// RoundStateUpdatedEvent represents the event triggered when a round's state is updated.
type RoundStateUpdatedEvent struct {
	RoundID int64             `json:"round_id"`
	State   models.RoundState `json:"state"`
}

// Topic returns the topic for the RoundStateUpdatedEvent.
func (e RoundStateUpdatedEvent) Topic() string {
	return "round.state.updated"
}

// UpdateRoundStateHandler handles the UpdateRoundState command.
type UpdateRoundStateHandler struct {
	roundDB    db.RoundDB
	messageBus watermill.Publisher
	logger     watermill.LoggerAdapter
}

// NewUpdateRoundStateHandler creates a new UpdateRoundStateHandler.
func NewUpdateRoundStateHandler(roundDB db.RoundDB, messageBus watermill.Publisher, logger watermill.LoggerAdapter) *UpdateRoundStateHandler {
	return &UpdateRoundStateHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
		logger:     logger,
	}
}

// Handle processes the UpdateRoundState command.
func (h *UpdateRoundStateHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd UpdateRoundStateRequest
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
