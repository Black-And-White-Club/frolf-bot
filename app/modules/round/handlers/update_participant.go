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

	// 1. Fetch the existing participant from the database using the new GetParticipant function
	participant, err := h.roundDB.GetParticipant(context.Background(), cmd.RoundID, cmd.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to get participant: %w", err)
	}

	// 2. Update the participant's fields based on the command data
	participant.Response = cmd.Response // Update the response (no need for a conditional check since it's required)
	if cmd.TagNumber != nil {
		participant.TagNumber = cmd.TagNumber
	}

	// 3. Update the participant's record in the database
	err = h.roundDB.UpdateParticipant(context.Background(), cmd.RoundID, participant)
	if err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	return nil
}
