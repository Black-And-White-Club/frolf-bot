package roundhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/Black-And-White-Club/tcr-bot/round/common"
	"github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	"github.com/ThreeDotsLabs/watermill/message"
)

type ScheduleRoundRequest struct {
	common.ScheduleRoundInput
}

func (ScheduleRoundRequest) CommandName() string {
	return "ScheduleRoundRequest"
}

type ScheduleRoundHandler struct {
	roundDB   rounddb.RoundDB
	converter converter.RoundConverter
	eventBus  *watermillutil.PubSub
}

func NewScheduleRoundHandler(roundDB rounddb.RoundDB, converter converter.RoundConverter, eventBus *watermillutil.PubSub) *ScheduleRoundHandler {
	return &ScheduleRoundHandler{
		roundDB:   roundDB,
		converter: converter,
		eventBus:  eventBus,
	}
}

func (h *ScheduleRoundHandler) Handler(msg *message.Message) error {
	var cmd ScheduleRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal ScheduleRoundRequest: %w", err)
	}

	if cmd.Title == "" {
		return errors.New("title is required")
	}

	modelInput := h.converter.ConvertScheduleRoundInputToModel(cmd.ScheduleRoundInput)

	round, err := h.roundDB.CreateRound(context.Background(), modelInput)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	// Publish a RoundScheduled event (you'll need to define this event)
	if err := h.eventBus.Publish(context.Background(), "RoundScheduled", &RoundScheduled{
		RoundID: round.ID,
		// ... other relevant data from the round
	}); err != nil {
		return fmt.Errorf("failed to publish RoundScheduled event: %w", err)
	}

	return nil
}
