package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCreatedHandler handles the UserCreatedEvent.
type UserCreatedHandler struct {
	eventBus      watermillutil.PubSuber
	leaderboardDB leaderboarddb.LeaderboardDB
}

// NewUserCreatedHandler creates a new UserCreatedHandler.
func NewUserCreatedHandler(eventBus watermillutil.PubSuber, leaderboardDB leaderboarddb.LeaderboardDB) *UserCreatedHandler {
	h := &UserCreatedHandler{
		eventBus:      eventBus,
		leaderboardDB: leaderboardDB,
	}

	// Subscribe to the UserCreatedEvent
	_, err := h.eventBus.Subscribe(context.Background(), TopicUserCreated) // Assign both return values
	if err != nil {
		fmt.Println("Error subscribing to UserCreatedEvent:", err)
	}

	return h
}

// UserCreatedEvent represents the event for a user being created.
type UserCreatedEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// Handle processes the UserCreatedEvent.
func (h *UserCreatedHandler) Handle(msg *message.Message) error {
	var event UserCreatedEvent
	if err := watermillutil.Marshaler.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal UserCreatedEvent: %w", err)
	}

	// Insert the tag and Discord ID into the leaderboard using leaderboardDB
	if err := h.leaderboardDB.InsertTagAndDiscordID(context.Background(), event.TagNumber, event.DiscordID); err != nil {
		return fmt.Errorf("failed to insert tag and Discord ID into leaderboard: %w", err)
	}

	return nil
}
