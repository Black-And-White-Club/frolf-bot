package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// FinalizeRoundHandler handles the FinalizeRound command.
type FinalizeRoundHandler struct {
	RoundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
}

// NewFinalizeRoundHandler creates a new FinalizeRoundHandler.
func NewFinalizeRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher) *FinalizeRoundHandler {
	return &FinalizeRoundHandler{
		RoundDB:    roundDB,
		messageBus: messageBus,
	}
}

// Handle processes the FinalizeRound command.
func (h *FinalizeRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	log.Println("Round finalize handler called")

	var cmd roundcommands.FinalizeRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal FinalizeRoundRequest: %w", err)
	}

	// Get the round from the database
	round, err := h.RoundDB.GetRound(ctx, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round is already finalized
	if round.State == rounddb.RoundStateFinalized {
		return fmt.Errorf("round is already finalized")
	}

	// Publish command to update the round state
	updateStateCmd := roundcommands.UpdateRoundStateRequest{
		RoundID: cmd.RoundID,
		State:   rounddb.RoundStateFinalized,
	}
	// Marshal the command into JSON
	updateStatePayload, err := json.Marshal(updateStateCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal UpdateRoundStateRequest: %w", err)
	}

	if err := h.messageBus.Publish(updateStateCmd.CommandName(), message.NewMessage(watermill.NewUUID(), updateStatePayload)); err != nil {
		return fmt.Errorf("failed to publish UpdateRoundStateRequest: %w", err)
	}

	return nil
}
