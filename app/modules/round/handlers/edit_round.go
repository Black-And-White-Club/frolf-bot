package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/Black-And-White-Club/tcr-bot/round/converter"
	"github.com/ThreeDotsLabs/watermill/message"
)

type EditRoundRequest struct {
	RoundID int64
	converter.EditRoundInput
}

func (EditRoundRequest) CommandName() string {
	return "EditRoundRequest"
}

type EditRoundHandler struct {
	roundDB   rounddb.RoundDB
	converter converter.RoundConverter
	eventBus  *watermillutil.PubSub
}

func NewEditRoundHandler(roundDB rounddb.RoundDB, converter converter.RoundConverter, eventBus *watermillutil.PubSub) *EditRoundHandler {
	return &EditRoundHandler{
		roundDB:   roundDB,
		converter: converter,
		eventBus:  eventBus,
	}
}

func (h *EditRoundHandler) Handler(msg *message.Message) error {
	var cmd EditRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal EditRoundRequest: %w", err)
	}

	modelInput := h.converter.ConvertEditRoundInputToModel(cmd.EditRoundInput)

	err := h.roundDB.UpdateRound(context.Background(), cmd.RoundID, modelInput)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	// Publish a RoundEdited event (you'll need to define this event)
	if err := h.eventBus.Publish(context.Background(), "RoundEdited", &RoundEdited{
		RoundID: cmd.RoundID,
		// ... other relevant data if needed
	}); err != nil {
		return fmt.Errorf("failed to publish RoundEdited event: %w", err)
	}

	return nil
}
