package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// ReceiveScoresHandler handles receiving sorted scores from the score module.
type ReceiveScoresHandler struct {
	eventBus watermillutil.PubSuber
}

// NewReceiveScoresHandler creates a new ReceiveScoresHandler.
func NewReceiveScoresHandler(eventBus watermillutil.PubSuber) *ReceiveScoresHandler {
	return &ReceiveScoresHandler{eventBus: eventBus}
}

// Handle receives the sorted scores and publishes an event.
func (h *ReceiveScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var entries []leaderboarddto.LeaderboardEntry
	if err := json.Unmarshal(msg.Payload, &entries); err != nil {
		return errors.Wrap(err, "failed to unmarshal leaderboard entries")
	}

	// Publish LeaderboardEntriesReceivedEvent
	event := LeaderboardEntriesReceivedEvent{Entries: entries}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal LeaderboardEntriesReceivedEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicLeaderboardEntriesReceived, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish LeaderboardEntriesReceivedEvent: %w", err)
	}

	return nil
}
