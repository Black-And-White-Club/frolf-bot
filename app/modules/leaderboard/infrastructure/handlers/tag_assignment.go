package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAssignment handles the TagAssignmentRequested event.
func (h *LeaderboardHandlers) HandleTagAssignment(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagAssignment",
		&leaderboardevents.TagAssignmentRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagAssignmentRequestedPayload := payload.(*leaderboardevents.TagAssignmentRequestedPayload)

			h.logger.Info("Received TagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagAssignmentRequestedPayload.UserID)),
				attr.Int("tag_number", int(*tagAssignmentRequestedPayload.TagNumber)),
			)

			// Call the service function to handle the event
			result, err := h.leaderboardService.TagAssignmentRequested(ctx, msg, *tagAssignmentRequestedPayload)
			if err != nil {
				h.logger.Error("Failed to handle TagAssignmentRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle TagAssignmentRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Error("Tag assignment failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.TagAssignmentFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			h.logger.Info("Tag assignment successful", attr.CorrelationIDFromMsg(msg))

			// Create success message to publish
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				result.Success,
				leaderboardevents.TagAssigned,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			// Publish TagAvailable event to User module
			tagAvailablePayload := &leaderboardevents.TagAvailablePayload{
				UserID:    result.Success.(*leaderboardevents.TagAssignedPayload).UserID,
				TagNumber: result.Success.(*leaderboardevents.TagAssignedPayload).TagNumber,
			}
			tagAvailableMsg, err := h.helpers.CreateResultMessage(
				msg,
				tagAvailablePayload,
				leaderboardevents.TagAvailable,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create TagAvailable message: %w", err)
			}

			return []*message.Message{successMsg, tagAvailableMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
