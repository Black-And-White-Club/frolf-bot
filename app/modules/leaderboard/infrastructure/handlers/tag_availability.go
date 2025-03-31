package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAvailabilityCheckRequested handles the TagAvailabilityCheckRequested event.
func (h *LeaderboardHandlers) HandleTagAvailabilityCheckRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagAvailabilityCheckRequested",
		&leaderboardevents.TagAvailabilityCheckRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagAvailabilityCheckRequestedPayload := payload.(*leaderboardevents.TagAvailabilityCheckRequestedPayload)

			h.logger.Info("Received TagAvailabilityCheckRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagAvailabilityCheckRequestedPayload.UserID)),
				attr.Int("tag_number", int(*tagAvailabilityCheckRequestedPayload.TagNumber)),
			)

			// Call the service function to handle the event
			result, failure, err := h.leaderboardService.CheckTagAvailability(ctx, *tagAvailabilityCheckRequestedPayload)
			if err != nil {
				h.logger.Error("Failed to handle TagAvailabilityCheckRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle TagAvailabilityCheckRequested event: %w", err)
			}

			if failure != nil {
				h.logger.Info("Tag availability check failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					failure,
					leaderboardevents.TagAvailableCheckFailure,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			h.logger.Info("Tag availability check successful", attr.CorrelationIDFromMsg(msg))

			// Create success message to publish
			if result.Available {
				h.logger.Info("Tag is available",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(result.UserID)),
					attr.Int("tag_number", int(*result.TagNumber)),
				)

				// Create success message
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result,
					leaderboardevents.TagAvailable,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			} else {
				h.logger.Info("Tag is not available",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(result.UserID)),
					attr.Int("tag_number", int(*result.TagNumber)),
				)

				// Create tag not available message
				tagNotAvailableMsg, err := h.helpers.CreateResultMessage(
					msg,
					&leaderboardevents.TagUnavailablePayload{
						UserID:    result.UserID,
						TagNumber: result.TagNumber,
					},
					leaderboardevents.TagUnavailable,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create tag not available message: %w", err)
				}

				return []*message.Message{tagNotAvailableMsg}, nil
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
