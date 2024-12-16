package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// SendScoresHandler handles sending the sorted scores to the leaderboard module.
type SendScoresHandler struct {
	eventBus watermillutil.PubSuber
}

// NewSendScoresHandler creates a new SendScoresHandler instance.
func NewSendScoresHandler(eventBus watermillutil.PubSuber) *SendScoresHandler {
	return &SendScoresHandler{
		eventBus: eventBus,
	}
}

// Handle sends the Discord IDs and Tag Numbers to the leaderboard module.
func (h *SendScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var scores []scoredb.Score
	if err := json.Unmarshal(msg.Payload, &scores); err != nil {
		return errors.Wrap(err, "failed to unmarshal sorted scores")
	}

	// Extract Discord IDs and Tag Numbers
	leaderboardData := make([]LeaderboardEntry, len(scores))
	for i, score := range scores {
		leaderboardData[i] = LeaderboardEntry{
			DiscordID: score.DiscordID,
			TagNumber: score.TagNumber,
		}
	}

	// Publish leaderboard data
	payload, err := json.Marshal(leaderboardData)
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard data: %w", err)
	}

	if err := h.eventBus.Publish(TopicUpdateLeaderboard, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish leaderboard data: %w", err)
	}

	return nil
}

// LeaderboardEntry represents the data sent to the leaderboard module.
type LeaderboardEntry struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}
