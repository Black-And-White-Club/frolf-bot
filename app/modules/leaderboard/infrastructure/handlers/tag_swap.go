package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagSwapRequested handles the TagSwapRequested event.
func (h *LeaderboardHandlers) HandleTagSwapRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagSwapRequested",
		&leaderboardevents.TagSwapRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagSwapRequestedPayload := payload.(*leaderboardevents.TagSwapRequestedPayload)

			h.logger.InfoContext(ctx, "Received TagSwapRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("requestor_id", string(tagSwapRequestedPayload.RequestorID)),
				attr.String("target_id", string(tagSwapRequestedPayload.TargetID)),
			)

			// Call the service function to handle the event
			result, err := h.leaderboardService.TagSwapRequested(ctx, *tagSwapRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle TagSwapRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle TagSwapRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Tag swap failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.TagSwapFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			} else if result.Success != nil {
				h.logger.InfoContext(ctx, "Tag swap successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.TagSwapProcessed,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			} else {
				// Handle the case where both Success and Failure are nil
				h.logger.ErrorContext(ctx, "Unexpected result from service: both success and failure are nil",
					attr.CorrelationIDFromMsg(msg),
				)
				return nil, fmt.Errorf("unexpected result from service")
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
