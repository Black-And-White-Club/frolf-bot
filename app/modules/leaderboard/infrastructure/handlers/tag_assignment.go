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

			h.logger.InfoContext(ctx, "Received TagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagAssignmentRequestedPayload.UserID)),
				attr.Int("tag_number", int(*tagAssignmentRequestedPayload.TagNumber)),
			)

			// Call the service function to handle the event
			result, err := h.leaderboardService.TagAssignmentRequested(ctx, *tagAssignmentRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle TagAssignmentRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle TagAssignmentRequested event: %w", err)
			}

			// Handle the result
			if result.Failure != nil {
				h.logger.ErrorContext(ctx, "Tag assignment failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardTagAssignmentFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Tag assignment successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardTagAssignmentSuccess,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				return []*message.Message{successMsg}, nil
			}

			// If neither Success nor Failure is set, return an error
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
