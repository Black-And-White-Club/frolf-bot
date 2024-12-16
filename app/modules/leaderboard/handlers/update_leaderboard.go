package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/services"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// UpdateLeaderboardHandler handles updating the leaderboard in the database.
type UpdateLeaderboardHandler struct {
	leaderboardDB             leaderboarddb.LeaderboardDB
	leaderboardUpdaterService leaderboardservice.LeaderboardUpdateService
}

// NewUpdateLeaderboardHandler creates a new UpdateLeaderboardHandler.
func NewUpdateLeaderboardHandler(leaderboardDB leaderboarddb.LeaderboardDB, leaderboardUpdaterService leaderboardservice.LeaderboardUpdateService) *UpdateLeaderboardHandler {
	return &UpdateLeaderboardHandler{
		leaderboardDB:             leaderboardDB,
		leaderboardUpdaterService: leaderboardUpdaterService,
	}
}

// Handle updates the leaderboard with the assigned tags.
func (h *UpdateLeaderboardHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event LeaderboardTagsAssignedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal LeaderboardTagsAssignedEvent")
	}

	// 1. Get the current leaderboard data
	currentLeaderboard, err := h.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current leaderboard: %w", err)
	}

	// 2. Update the leaderboard data with new tags
	updatedLeaderboardData := h.leaderboardUpdaterService.UpdateLeaderboardData(currentLeaderboard.LeaderboardData, event.Entries)

	// 3. Update the leaderboard within a transaction
	err = h.leaderboardDB.UpdateLeaderboardWithTransaction(ctx, updatedLeaderboardData)
	if err != nil {
		return fmt.Errorf("failed to update leaderboard with transaction: %w", err)
	}

	return nil
}
