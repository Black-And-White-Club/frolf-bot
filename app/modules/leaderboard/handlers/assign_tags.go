package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/services"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// AssignTagsHandler assigns tags to leaderboard entries.
type AssignTagsHandler struct {
	eventBus             watermillutil.PubSuber
	tagAssignmentService leaderboardservice.TagAssignmentService
}

// NewAssignTagsHandler creates a new AssignTagsHandler.
func NewAssignTagsHandler(eventBus watermillutil.PubSuber, tagAssignmentService leaderboardservice.TagAssignmentService) *AssignTagsHandler {
	return &AssignTagsHandler{
		eventBus:             eventBus,
		tagAssignmentService: tagAssignmentService,
	}
}

// Handle assigns tags to the received leaderboard entries.
func (h *AssignTagsHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event LeaderboardEntriesReceivedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaderboardEntriesReceivedEvent")
	}

	// Use the service to assign tags
	assignedEntries, err := h.tagAssignmentService.AssignTags(ctx, event.Entries)
	if err != nil {
		return fmt.Errorf("failed to assign tags: %w", err)
	}

	// Publish LeaderboardTagsAssignedEvent
	assignedEvent := LeaderboardTagsAssignedEvent{Entries: assignedEntries}
	payload, err := json.Marshal(assignedEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal LeaderboardTagsAssignedEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicLeaderboardTagsAssigned, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish LeaderboardTagsAssignedEvent: %w", err)
	}

	return nil
}
