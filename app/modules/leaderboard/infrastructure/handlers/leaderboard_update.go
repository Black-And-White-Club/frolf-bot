package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleLeaderboardUpdateRequested",
		&leaderboardevents.LeaderboardUpdateRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			leaderboardUpdateRequestedPayload := payload.(*leaderboardevents.LeaderboardUpdateRequestedPayload)

			// Create convenient variables for frequently used fields
			roundID := leaderboardUpdateRequestedPayload.RoundID
			sortedParticipantTags := leaderboardUpdateRequestedPayload.SortedParticipantTags

			h.logger.Info("Received LeaderboardUpdateRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.Any("sorted_participant_tags", sortedParticipantTags),
			)

			// Call the service function to update the leaderboard
			result, err := h.leaderboardService.UpdateLeaderboard(ctx, msg, roundID, sortedParticipantTags)
			if err != nil {
				h.logger.Error("Failed to update leaderboard",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to update leaderboard: %w", err)
			}

			if result.Failure != nil {
				h.logger.Error("Leaderboard update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardUpdateFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			h.logger.Info("Leaderboard updated successfully", attr.CorrelationIDFromMsg(msg))

			// Create success message to publish
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				result.Success,
				leaderboardevents.LeaderboardUpdated,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			return []*message.Message{successMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
