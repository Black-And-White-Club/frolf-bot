// In app/modules/round/handlers/update_participant_handler.go

package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UpdateParticipantHandler handles the UpdateParticipantRequest command.
type UpdateParticipantHandler struct {
	roundDB  rounddb.RoundDB
	eventBus watermillutil.Publisher
}

// NewUpdateParticipantHandler creates a new UpdateParticipantHandler.
func NewUpdateParticipantHandler(roundDB rounddb.RoundDB, eventBus watermillutil.Publisher) *UpdateParticipantHandler {
	return &UpdateParticipantHandler{
		roundDB:  roundDB,
		eventBus: eventBus,
	}
}

// Handle processes the UpdateParticipantRequest command.
func (h *UpdateParticipantHandler) Handle(msg *message.Message) error {
	var cmd roundcommands.UpdateParticipantRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal UpdateParticipantRequest: %w", err)
	}

	// 1. Construct a Participant with the updated data
	participant := rounddb.Participant{
		DiscordID: cmd.DiscordID,
		// ... set other fields based on the command data ...
	}

	// 2. Update the participant's record in the database
	err := h.roundDB.UpdateParticipant(context.Background(), cmd.RoundID, participant)
	if err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	// 3. (Optional) Publish an event to indicate the participant was updated
	// ...

	return nil
}
