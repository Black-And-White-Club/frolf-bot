// In app/modules/round/handlers/tag_number_response.go

package roundhandlers

import (
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetTagNumberResponseHandler handles the GetTagNumberResponse event.
type GetTagNumberResponseHandler struct {
	roundDB  rounddb.RoundDB
	eventBus watermillutil.Publisher
}

// NewGetTagNumberResponseHandler creates a new GetTagNumberResponseHandler.
func NewGetTagNumberResponseHandler(roundDB rounddb.RoundDB, eventBus watermillutil.Publisher) *GetTagNumberResponseHandler {
	return &GetTagNumberResponseHandler{
		roundDB:  roundDB,
		eventBus: eventBus,
	}
}

// Handle processes the GetTagNumberResponse event.
func (h *GetTagNumberResponseHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the GetTagNumberResponse
	var response GetTagNumberResponse // Use the correct event struct
	if err := json.Unmarshal(msg.Payload, &response); err != nil {
		return fmt.Errorf("failed to unmarshal GetTagNumberResponse: %w", err)
	}

	// 2. Publish an UpdateParticipantRequest
	updateParticipantCmd := roundcommands.UpdateParticipantRequest{
		Input: rounddto.UpdateParticipantResponseInput{ // Use the DTO
			RoundID:   response.RoundID,
			DiscordID: response.DiscordID,
			TagNumber: response.TagNumber,
		},
	}
	payload, err := json.Marshal(updateParticipantCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal UpdateParticipantRequest: %w", err)
	}
	if err := h.eventBus.Publish(updateParticipantCmd.CommandName(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish UpdateParticipantRequest: %w", err)
	}

	return nil
}
