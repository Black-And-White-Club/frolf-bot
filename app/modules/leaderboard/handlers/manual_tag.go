package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	leaderboardservices "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/services"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// ManualTagAssignmentHandler handles the ManualTagAssignmentRequest command.
type ManualTagAssignmentHandler struct {
	eventBus           watermillutil.PubSuber
	queryService       leaderboardqueries.QueryService
	leaderboardService leaderboardservices.LeaderboardService
}

// NewManualTagAssignmentHandler creates a new ManualTagAssignmentHandler.
func NewManualTagAssignmentHandler(eventBus watermillutil.PubSuber, queryService leaderboardqueries.QueryService, leaderboardService leaderboardservices.LeaderboardService) *ManualTagAssignmentHandler {
	return &ManualTagAssignmentHandler{
		eventBus:           eventBus,
		queryService:       queryService,
		leaderboardService: leaderboardService,
	}
}

// Handle processes the ManualTagAssignmentRequest command.
func (h *ManualTagAssignmentHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd leaderboardcommands.ManualTagAssignmentRequest
	if err := watermillutil.Marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal ManualTagAssignmentRequest")
	}

	// Check if the tag is taken using the query service
	isTaken, err := h.queryService.IsTagTaken(ctx, cmd.TagNumber)
	if err != nil {
		return fmt.Errorf("failed to check if tag is taken: %w", err)
	}

	if isTaken {
		// Tag is taken, publish a TagSwapRequest event
		currentHolderID, err := h.queryService.GetTagHolder(ctx, cmd.TagNumber)
		if err != nil {
			return fmt.Errorf("failed to get tag holder: %w", err)
		}

		// Get the leaderboard data using GetLeaderboard
		leaderboard, err := h.queryService.GetLeaderboard(ctx)
		if err != nil {
			return fmt.Errorf("failed to get leaderboard: %w", err)
		}

		// Find the tag number of the current holder
		var targetTag int
		for tag, discordID := range leaderboard.LeaderboardData {
			if discordID == currentHolderID {
				targetTag = tag
				break
			}
		}

		// Publish TagSwapRequest event to trigger the TagSwapHandler
		tagSwapEvent := &TagSwapEvent{
			DiscordID: cmd.DiscordID,
			UserTag:   cmd.TagNumber,
			TargetTag: targetTag, // Use the retrieved target tag number
		}

		payload, err := json.Marshal(tagSwapEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal TagSwapEvent: %w", err)
		}

		if err := h.eventBus.Publish(TagSwapRequest, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
			return fmt.Errorf("failed to publish TagSwapEvent: %w", err)
		}

	} else {
		// Tag is available, publish an event to trigger AssignTagsHandler
		assignTagEvent := &AssignTagEvent{
			DiscordID: cmd.DiscordID,
			TagNumber: cmd.TagNumber,
		}

		payload, err := json.Marshal(assignTagEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal AssignTagEvent: %w", err)
		}

		if err := h.eventBus.Publish(TopicAssignTag, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
			return fmt.Errorf("failed to publish AssignTagEvent: %w", err)
		}
	}

	return nil
}
