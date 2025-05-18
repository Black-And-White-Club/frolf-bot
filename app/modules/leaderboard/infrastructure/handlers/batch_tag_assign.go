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
			// --- IMPORTANT: Check for error from service first ---
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

				// If the service also returned a failure payload in the result struct,
				// perform a type assertion and use its reason if available.
				// This check is secondary to the non-nil error check.
				if result.Failure != nil {
					// --- Perform type assertion here ---
					if serviceFailurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload); ok {
						failurePayload.Reason = serviceFailurePayload.Reason
						// Optionally copy other fields from serviceFailurePayload if needed
					} else {
						// Log a warning if result.Failure was not the expected type
						h.logger.WarnContext(ctx, "Service returned non-nil result.Failure, but it was not the expected type",
							attr.CorrelationIDFromMsg(msg),
							attr.Any("actual_type", fmt.Sprintf("%T", result.Failure)),
						)
						// Keep the default reason from the service error
					}
				}

				// --- Pass a pointer to the failurePayload to CreateResultMessage ---
				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					&failurePayload, // Pass the address of the struct
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				)
				if errMsg != nil {
					// If we fail to create the failure message, return the original service error
					// to Watermill for potential retry/dead-lettering.
					return nil, fmt.Errorf("failed to create failure message after service error: %w", errMsg)
				}

				// Return the failure message to be published.
				// Return nil error to Watermill so it doesn't retry this handler execution,
				// as we have successfully processed the failure by publishing a failure event.
				return []*message.Message{failureMsg}, nil
			}

			// If there was no error from the service, check for success or failure payloads in the result struct.
			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Batch tag assignment failed according to service (no service error returned)",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure), // result.Failure is already a pointer here if service returns it
				)

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					result.Failure, // Pass the pointer from the service result
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
					result.Success, // Pass the pointer from the service result
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
