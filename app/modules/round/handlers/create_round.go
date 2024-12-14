package roundhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CreateRoundHandler handles the CreateRoundRequest command.
type CreateRoundHandler struct {
	roundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
}

// NewCreateRoundHandler creates a new CreateRoundHandler.
func NewCreateRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher) *CreateRoundHandler {
	return &CreateRoundHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
	}
}

// Handler processes the CreateRoundRequest command.
func (h *CreateRoundHandler) Handler(msg *message.Message) error {
	var cmd roundcommands.CreateRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal CreateRoundRequest: %w", err)
	}

	if cmd.Input.Title == "" {
		return errors.New("title is required")
	}

	round, err := h.roundDB.CreateRound(context.Background(), cmd.Input)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	// Publish a RoundCreated event
	event := RoundCreatedEvent{
		RoundID: round.ID,
		Input:   cmd.Input,
		// ... other relevant data from the round ...
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundCreatedEvent: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish RoundCreatedEvent: %w", err)
	}

	return nil
}
