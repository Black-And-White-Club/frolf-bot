package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetLeaderboardRequest",
		&leaderboardevents.GetLeaderboardRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.logger.InfoContext(ctx, "Received GetLeaderboardRequest event",
				attr.CorrelationIDFromMsg(msg),
			)

			// Call the service function to get the leaderboard, propagate guildID
			payloadTyped := payload.(*leaderboardevents.GetLeaderboardRequestedPayloadV1)
			result, err := h.leaderboardService.GetLeaderboard(ctx, payloadTyped.GuildID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to get leaderboard",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to get leaderboard: %w", err)
			}

			if result.Failure != nil {
				h.logger.ErrorContext(ctx, "Get leaderboard failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.GetLeaderboardFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Get leaderboard successful",
					attr.CorrelationIDFromMsg(msg),
				)

				// Create success message to publish
				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.GetLeaderboardResponseV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Success nor Failure is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from GetLeaderboard",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleRoundGetTagRequest handles the RoundTagLookupRequest event.
func (h *LeaderboardHandlers) HandleRoundGetTagRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundGetTagRequest",
		&sharedevents.RoundTagLookupRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagLookupRequestPayload := payload.(*sharedevents.RoundTagLookupRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received RoundTagLookupRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagLookupRequestPayload.UserID)),
				attr.RoundID("round_id", tagLookupRequestPayload.RoundID),
				attr.String("response", string(tagLookupRequestPayload.Response)),
				attr.Any("joined_late", tagLookupRequestPayload.JoinedLate),
			)

			result, err := h.leaderboardService.RoundGetTagByUserID(ctx, sharedtypes.GuildID(tagLookupRequestPayload.GuildID), *tagLookupRequestPayload)
			// ServiceWrapper returns error for unexpected system errors.
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during RoundGetTagByUserID service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				// Return the error to Watermill for retry/dead-lettering.
				return nil, fmt.Errorf("failed during RoundGetTagByUserID service call: %w", err)
			}

			// Handle business outcomes based on the result.
			if result.Success != nil {
				responsePayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayloadV1)
				if !ok {
					err := fmt.Errorf("unexpected success payload type from RoundGetTagByUserID: expected *sharedevents.RoundTagLookupResultPayloadV1, got %T", result.Success)
					h.logger.ErrorContext(ctx, "Unexpected success payload type from service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, err
				}

				eventType := sharedevents.RoundTagLookupNotFoundV1 // Default to not found
				eventName := "RoundTagLookupNotFoundV1"

				if responsePayload.Found {
					eventType = sharedevents.RoundTagLookupFoundV1
					eventName = "RoundTagLookupFoundV1"
					h.logger.InfoContext(ctx, "Tag lookup successful: Tag found",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
						attr.Int("tag_number", int(*responsePayload.TagNumber)),
					)
				} else {
					h.logger.InfoContext(ctx, "Tag lookup completed: Tag not found (Business Outcome)",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
					)
				}

				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					responsePayload,
					eventType,
				)
				if err != nil {
					h.logger.ErrorContext(ctx, fmt.Sprintf("Failed to create %s message", eventName),
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
					)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				h.logger.InfoContext(ctx, fmt.Sprintf("Publishing %s message", eventName),
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", successMsg.UUID),
					attr.String("topic", eventType),
				)
				return []*message.Message{successMsg}, nil

			} else if result.Failure != nil {
				// Handle business failure (e.g., No active leaderboard)
				failurePayload, ok := result.Failure.(*sharedevents.RoundTagLookupFailedPayloadV1)
				if !ok {
					err := fmt.Errorf("unexpected failure payload type from RoundGetTagByUserID: expected *sharedevents.RoundTagLookupFailedPayloadV1, got %T", result.Failure)
					h.logger.ErrorContext(ctx, "Unexpected failure payload type from service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Failure)),
					)
					return nil, err
				}

				h.logger.InfoContext(ctx, "RoundGetTagByUserID service returned business failure",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failurePayload.Reason),
				)

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					failurePayload,
					leaderboardevents.GetTagNumberFailedV1,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after business failure",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				h.logger.InfoContext(ctx, "Publishing GetTagNumberFailedV1 message due to business failure",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", failureMsg.UUID),
					attr.String("topic", leaderboardevents.GetTagNumberFailedV1),
				)
				return []*message.Message{failureMsg}, nil

			} else if result.Error != nil {
				// Handle unexpected system error returned within the result struct
				h.logger.ErrorContext(ctx, "RoundGetTagByUserID service returned system error within result",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(result.Error),
				)

				failurePayload := sharedevents.RoundTagLookupFailedPayloadV1{
					UserID:  tagLookupRequestPayload.UserID,
					RoundID: tagLookupRequestPayload.RoundID,
					Reason:  result.Error.Error(),
				}

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					failurePayload,
					leaderboardevents.GetTagNumberFailedV1,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after service system error",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				h.logger.InfoContext(ctx, "Publishing GetTagNumberFailedV1 message due to system error",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", failureMsg.UUID),
					attr.String("topic", leaderboardevents.GetTagNumberFailedV1),
				)
				return []*message.Message{failureMsg}, nil

			} else {
				// Unexpected scenario where service returned neither success, failure, nor error in result
				err := fmt.Errorf("RoundGetTagByUserID service returned unexpected nil result fields")
				h.logger.ErrorContext(ctx, "Unexpected nil result fields from service",
					attr.CorrelationIDFromMsg(msg),
				)
				return nil, err // Return non-nil error to Watermill
			}
		},
	)

	return wrappedHandler(msg)
}

// HandleGetTagByUserIDRequest handles the GetTagByUserIDRequest event.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(msg *message.Message) ([]*message.Message, error) {
	// Updated expected input payload type
	wrappedHandler := h.handlerWrapper("HandleGetTagByUserIDRequest",
		&sharedevents.DiscordTagLookupRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Cast the payload to the expected input type
			tagNumberRequestPayload, ok := payload.(*sharedevents.DiscordTagLookupRequestedPayloadV1)
			if !ok {
				err := fmt.Errorf("unexpected payload type for HandleGetTagByUserIDRequest: expected *sharedevents.DiscordTagLookupRequestPayload, got %T", payload)
				h.logger.ErrorContext(ctx, "Unexpected payload type in handler",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_type", fmt.Sprintf("%T", payload)),
				)
				// Return non-nil error to Watermill for retry/dead-lettering.
				return nil, fmt.Errorf("handler payload type assertion failed: %w", err)
			}

			h.logger.InfoContext(ctx, "Received DiscordTagLookUpByUserIDRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagNumberRequestPayload.UserID)),
			)

			// Call the service method with the UserID and GuildID
			result, err := h.leaderboardService.GetTagByUserID(ctx, sharedtypes.GuildID(tagNumberRequestPayload.GuildID), tagNumberRequestPayload.UserID)
			// Check for system errors returned directly by the service call first.
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during GetTagByUserID service call (system error)",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				// Return the error to Watermill for retry/dead-lettering.
				return nil, fmt.Errorf("failed during GetTagByUserID service call: %w", err)
			}

			// If no system error from the service call, handle business outcomes based on the result struct.
			if result.Success != nil {
				// Cast to the success type returned by the service
				successPayload, ok := result.Success.(*sharedevents.DiscordTagLookupResultPayloadV1)
				if !ok {
					err := fmt.Errorf("unexpected success payload type from GetTagByUserID service: expected *sharedevents.DiscordTagLookupResultPayloadV1, got %T", result.Success)
					h.logger.ErrorContext(ctx, "Unexpected success payload type from service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					// Return non-nil error to Watermill for retry/dead-lettering.
					return nil, fmt.Errorf("service success payload type assertion failed: %w", err)
				}

				// Determine which event type to use based on whether the tag was found
				eventType := sharedevents.DiscordTagLookupNotFoundV1

				// The responsePayload is the same as the successPayload from the service in this case
				responsePayload := successPayload

				if responsePayload.Found && responsePayload.TagNumber != nil {
					eventType = sharedevents.DiscordTagLookupSucceededV1

					h.logger.InfoContext(ctx, "Tag lookup successful: Tag found",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
						attr.Int("tag_number", int(*responsePayload.TagNumber)),
					)
				} else {
					// Tag not found is a successful business outcome, just Found is false
					h.logger.InfoContext(ctx, "Tag lookup completed: Tag not found (Business Outcome)",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
					)
				}

				// Create message with appropriate event type and the handler's response payload
				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					responsePayload,
					eventType,
				)
				if err != nil {
					h.logger.ErrorContext(ctx, fmt.Sprintf("Failed to create %s message", eventType),
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
					)
					// Return non-nil error to Watermill for retry/dead-lettering.
					return nil, fmt.Errorf("failed to create success/not found message: %w", err)
				}

				h.logger.InfoContext(ctx, fmt.Sprintf("Publishing %s message", eventType),
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", successMsg.UUID),
					attr.String("topic", eventType),
				)
				// Return the message slice and nil error, indicating successful handler processing
				return []*message.Message{successMsg}, nil

			} else if result.Failure != nil {
				// Handle business failure (e.g., No active leaderboard)
				// Cast to the actual failure type returned by the service
				failurePayload, ok := result.Failure.(*sharedevents.DiscordTagLookupFailedPayloadV1)
				if !ok {
					err := fmt.Errorf("unexpected failure payload type from GetTagByUserID service: expected *sharedevents.DiscordTagLookupFailedPayloadV1, got %T", result.Failure)
					h.logger.ErrorContext(ctx, "Unexpected failure payload type from service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Failure)),
					)
					// Return non-nil error to Watermill for retry/dead-lettering.
					return nil, fmt.Errorf("service failure payload type assertion failed: %w", err)
				}

				h.logger.InfoContext(ctx, "GetTagByUserID service returned business failure",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failurePayload.Reason),
				)

				// Use the failure payload directly from the service
				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					failurePayload,
					sharedevents.DiscordTagLookupFailedV1,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after business failure",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					// Return non-nil error to Watermill for retry/dead-lettering.
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				h.logger.InfoContext(ctx, "Publishing DiscordTagLookupFailedV1 message due to business failure",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", failureMsg.UUID),
					attr.String("topic", sharedevents.DiscordTagLookupFailedV1),
				)
				// Return the message slice and nil error, indicating successful handler processing
				return []*message.Message{failureMsg}, nil

			} else if result.Error != nil {
				// Handle unexpected system error returned within the result struct
				h.logger.ErrorContext(ctx, "GetTagByUserID service returned system error within result",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(result.Error),
				)

				// Create the handler's failure payload for a system error
				handlerFailurePayload := sharedevents.DiscordTagLookupFailedPayloadV1{
					UserID: tagNumberRequestPayload.UserID,
					Reason: result.Error.Error(),
				}

				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					handlerFailurePayload,
					sharedevents.DiscordTagLookupFailedV1,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after service system error",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					// Return non-nil error to Watermill for retry/dead-lettering.
					return nil, fmt.Errorf("failed to create failure message after system error: %w", errMsg)
				}

				h.logger.InfoContext(ctx, "Publishing DiscordTagLookupFailedV1 message due to system error",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", failureMsg.UUID),
					attr.String("topic", sharedevents.DiscordTagLookupFailedV1),
				)
				// Return the message slice and nil error, indicating successful handler processing
				return []*message.Message{failureMsg}, nil

			} else {
				// Unexpected scenario where service returned neither success, failure, nor error.
				err := fmt.Errorf("GetTagByUserID service returned unexpected nil result fields")
				h.logger.ErrorContext(ctx, "Unexpected nil result fields from service",
					attr.CorrelationIDFromMsg(msg),
				)
				// Return non-nil error to Watermill for retry/dead-lettering.
				return nil, fmt.Errorf("service returned unexpected nil result: %w", err)
			}
		},
	)

	return wrappedHandler(msg)
}
