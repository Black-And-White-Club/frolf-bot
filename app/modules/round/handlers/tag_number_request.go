// In app/modules/round/handlers/get_tag_number_request_handler.go

package roundhandlers

import (
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
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
	var request roundcommands.GetTagNumberRequest // Declare a variable to hold the unmarshaled request
	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		return fmt.Errorf("failed to unmarshal GetTagNumberRequest: %w", err)
	}

	// Create a new message with the DiscordID as the payload
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal GetTagNumberRequest: %w", err)
	}
	newMsg := message.NewMessage(watermill.NewUUID(), payload)

	if err := h.eventBus.Publish("leaderboard.get.tag.number.request", newMsg); err != nil {
		return fmt.Errorf("failed to forward GetTagNumberRequest: %w", err)
	}
	return nil
}
