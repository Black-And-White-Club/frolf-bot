package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboardservices "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/services"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// ManualTagAssignmentHandler handles the ManualTagAssignmentRequest command.
type ManualTagAssignmentHandler struct {
	eventBus             watermillutil.PubSuber
	tagAssignmentService leaderboardservices.TagAssignmentService
}

// NewManualTagAssignmentHandler creates a new ManualTagAssignmentHandler.
func NewManualTagAssignmentHandler(eventBus watermillutil.PubSuber, tagAssignmentService leaderboardservices.TagAssignmentService) *ManualTagAssignmentHandler {
	return &ManualTagAssignmentHandler{
		eventBus:             eventBus,
		tagAssignmentService: tagAssignmentService,
	}
}

// Handle processes the ManualTagAssignmentRequest command.
func (h *ManualTagAssignmentHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd leaderboardcommands.ManualTagAssignmentRequest
	marshaler := watermillutil.Marshaler
	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal ManualTagAssignmentRequest")
	}

	// Check if the tag is taken
	isTaken, err := h.tagAssignmentService.IsTagTaken(ctx, cmd.TagNumber)
	if err != nil {
		return fmt.Errorf("failed to check if tag is taken: %w", err)
	}

	if isTaken {
		// Tag is taken, publish an event to notify the user and ask about swapping
		// ... (Publish TagTakenEvent with swap option)
	} else {
		// Tag is available, assign it to the user
		if err := h.tagAssignmentService.AssignTagToUser(ctx, cmd.DiscordID, cmd.TagNumber); err != nil {
			return fmt.Errorf("failed to assign tag to user: %w", err)
		}

		// Publish TagAssignedEvent
		// ...
	}

	return nil
}
