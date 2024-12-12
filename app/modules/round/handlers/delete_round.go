package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type DeleteRoundRequest struct {
	RoundID int64
}

func (DeleteRoundRequest) CommandName() string {
	return "DeleteRoundRequest"
}

type DeleteRoundHandler struct {
	roundDB  rounddb.RoundDB
	eventBus *watermillutil.PubSub
}

func NewDeleteRoundHandler(roundDB rounddb.RoundDB, eventBus *watermillutil.PubSub) *DeleteRoundHandler {
	return &DeleteRoundHandler{
		roundDB:  roundDB,
		eventBus: eventBus,
	}
}

func (h *DeleteRoundHandler) Handler(msg *message.Message) error {
	var cmd DeleteRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal DeleteRoundRequest: %w", err)
	}

	err := h.roundDB.DeleteRound(context.Background(), cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	// Publish a RoundDeleted event (you'll need to define this event)
	if err := h.eventBus.Publish(context.Background(), "RoundDeleted", &RoundDeleted{
		RoundID: cmd.RoundID,
		// ... other relevant data if needed
	}); err != nil {
		return fmt.Errorf("failed to publish RoundDeleted event: %w", err)
	}

	return nil
}
