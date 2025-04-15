package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleBatchTagAssignmentRequested handles the BatchTagAssignmentRequested event.
func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleBatchTagAssignmentRequested",
		&leaderboardevents.BatchTagAssignmentRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			batchTagAssignmentRequestedPayload := payload.(*leaderboardevents.BatchTagAssignmentRequestedPayload)

			h.logger.InfoContext(ctx, "Received BatchTagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("batch_id", batchTagAssignmentRequestedPayload.BatchID),
				attr.String("requesting_user", string(batchTagAssignmentRequestedPayload.RequestingUserID)),
				attr.Int("assignment_count", len(batchTagAssignmentRequestedPayload.Assignments)),
			)

			// Call the service function to handle the event
			result, err := h.leaderboardService.BatchTagAssignmentRequested(ctx, *batchTagAssignmentRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle BatchTagAssignmentRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle BatchTagAssignmentRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Batch tag assignment failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Batch tag assignment successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardBatchTagAssigned,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from BatchTagAssignmentRequested service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
