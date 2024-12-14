// In app/modules/round/handlers/get_tag_number_request_handler.go

package roundhandlers

import (
	"fmt"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetTagNumberRequestHandler handles the GetTagNumberRequest event by forwarding it to the leaderboard module.
type GetTagNumberRequestHandler struct {
	eventBus watermillutil.Publisher
}

// NewGetTagNumberRequestHandler creates a new GetTagNumberRequestHandler.
func NewGetTagNumberRequestHandler(eventBus watermillutil.Publisher) *GetTagNumberRequestHandler {
	return &GetTagNumberRequestHandler{
		eventBus: eventBus,
	}
}

// Handle forwards the GetTagNumberRequest to the leaderboard module.
func (h *GetTagNumberRequestHandler) Handle(msg *message.Message) error {
	if err := h.eventBus.Publish("leaderboard.get.tag.number.request", msg); err != nil {
		return fmt.Errorf("failed to forward GetTagNumberRequest: %w", err)
	}
	return nil
}
