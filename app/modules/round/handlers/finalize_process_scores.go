package roundhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/Black-And-White-Club/tcr-bot/round/common"
	"github.com/Black-And-White-Club/tcr-bot/round/converter"
	"github.com/ThreeDotsLabs/watermill/message"
)

type FinalizeAndProcessScoresRequest struct {
	RoundID int64
}

func (FinalizeAndProcessScoresRequest) CommandName() string {
	return "FinalizeAndProcessScoresRequest"
}

type FinalizeAndProcessScoresHandler struct {
	roundDB   rounddb.RoundDB
	converter converter.RoundConverter
	eventBus  *watermillutil.PubSub
}

func NewFinalizeAndProcessScoresHandler(roundDB rounddb.RoundDB, converter converter.RoundConverter, eventBus *watermillutil.PubSub) *FinalizeAndProcessScoresHandler {
	return &FinalizeAndProcessScoresHandler{
		roundDB:   roundDB,
		converter: converter,
		eventBus:  eventBus,
	}
}

func (h *FinalizeAndProcessScoresHandler) Handler(msg *message.Message) error {
	var cmd FinalizeAndProcessScoresRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal FinalizeAndProcessScoresRequest: %w", err)
	}

	round, err := h.roundDB.GetRound(context.Background(), cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}
	if round == nil {
		return errors.New("round not found")
	}

	if round.State == common.RoundStateFinalized {
		return nil // Already finalized
	}

	// ... (Logic to process scores - you'll need to adapt this from your existing code)

	if err := h.roundDB.UpdateRoundState(context.Background(), cmd.RoundID, common.RoundStateFinalized); err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// Publish a RoundFinalized event (you'll need to define this event)
	if err := h.eventBus.Publish(context.Background(), "RoundFinalized", &RoundFinalized{
		RoundID: cmd.RoundID,
		// ... other relevant data if needed
	}); err != nil {
		return fmt.Errorf("failed to publish RoundFinalized event: %w", err)
	}

	return nil
}
