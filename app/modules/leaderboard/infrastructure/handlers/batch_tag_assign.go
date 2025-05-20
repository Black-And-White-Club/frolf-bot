package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleBatchTagAssignmentRequested handles the BatchTagAssignmentRequested event.
// It consumes the sharedevents.LeaderboardBatchTagAssignmentRequested event,
// calls the LeaderboardService to perform the batch assignment,
// and publishes either a leaderboardevents.BatchTagAssigned or leaderboardevents.BatchTagAssignmentFailed event.
func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(msg *message.Message) ([]*message.Message, error) {
	// Use the handler wrapper for common logic
	wrappedHandler := h.handlerWrapper(
		"HandleBatchTagAssignmentRequested",
		&sharedevents.BatchTagAssignmentRequestedPayload{}, // Expecting sharedevents.BatchTagAssignmentRequestedPayload
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			batchTagAssignmentRequestedPayload := payload.(*sharedevents.BatchTagAssignmentRequestedPayload)

			h.logger.InfoContext(ctx, "Received BatchTagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("batch_id", batchTagAssignmentRequestedPayload.BatchID),
				attr.String("requesting_user", string(batchTagAssignmentRequestedPayload.RequestingUserID)),
				attr.Int("assignment_count", len(batchTagAssignmentRequestedPayload.Assignments)),
			)

			result, err := h.leaderboardService.BatchTagAssignmentRequested(ctx, *batchTagAssignmentRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Service failed to handle BatchTagAssignmentRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)

				// Create a failure payload struct.
				failurePayload := leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: batchTagAssignmentRequestedPayload.RequestingUserID,
					BatchID:          batchTagAssignmentRequestedPayload.BatchID,
					Reason:           err.Error(), // Default reason is the error message from the service call
				}

				if result.Failure != nil {
					if serviceFailurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload); ok {
						failurePayload.Reason = serviceFailurePayload.Reason
					} else {
						h.logger.WarnContext(ctx, "Service returned non-nil result.Failure, but it was not the expected type",
							attr.CorrelationIDFromMsg(msg),
							attr.Any("actual_type", fmt.Sprintf("%T", result.Failure)),
						)
					}
				}

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					&failurePayload, // Pass the address of the struct
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message after service error: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Batch tag assignment failed according to service (no service error returned)",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
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
				h.logger.InfoContext(ctx, "Batch tag assignment successful according to service", attr.CorrelationIDFromMsg(msg))

				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardBatchTagAssigned, // Publish to the leaderboardevents success topic
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// This case should ideally not happen if the service always returns Success or Failure when err is nil.
			h.logger.ErrorContext(ctx, "Service returned result with neither Success nor Failure payload set, and no error",
				attr.CorrelationIDFromMsg(msg),
			)
			// Return an error to Watermill so it might retry or move to dead-letter queue.
			return nil, fmt.Errorf("unexpected result from service: neither success nor failure payload set and no error")
		},
	)

	return wrappedHandler(msg)
}
