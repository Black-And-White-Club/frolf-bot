package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetLeaderboardRequest",
		&leaderboardevents.GetLeaderboardRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.logger.InfoContext(ctx, "Received GetLeaderboardRequest event",
				attr.CorrelationIDFromMsg(msg),
			)

			// Call the service function to get the leaderboard
			result, err := h.leaderboardService.GetLeaderboard(ctx)
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
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.GetLeaderboardFailed,
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
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.GetLeaderboardResponse,
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
		&sharedevents.RoundTagLookupRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagLookupRequestPayload := payload.(*sharedevents.RoundTagLookupRequestPayload)

			h.logger.InfoContext(ctx, "Received RoundTagLookupRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagLookupRequestPayload.UserID)),
				attr.RoundID("round_id", tagLookupRequestPayload.RoundID),
				attr.String("response", string(tagLookupRequestPayload.Response)),
				attr.Any("joined_late", tagLookupRequestPayload.JoinedLate),
			)

			// Call the service function to get the tag by userID.
			result, err := h.leaderboardService.RoundGetTagByUserID(ctx, *tagLookupRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during GetTagByUserID service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed during GetTagByUserID service call: %w", err)
			}

			// Assert the service result to the shared result payload.
			responsePayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayload)
			if !ok {
				err := fmt.Errorf("unexpected success payload type from GetTagByUserID: expected *roundevents.RoundTagLookupResultPayload, got %T", result.Success)
				h.logger.ErrorContext(ctx, "Unexpected success payload type from service",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
				)
				return nil, err
			}

			// Determine the event type based on the result.
			var eventType string
			var eventName string

			if responsePayload.Found && responsePayload.Error == "" {
				eventType = sharedevents.RoundTagLookupFound
				eventName = "RoundTagLookupFound"
				h.logger.InfoContext(ctx, "Tag lookup successful: Tag found",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(responsePayload.UserID)),
					attr.Int("tag_number", int(*responsePayload.TagNumber)),
					attr.String("original_response", string(responsePayload.OriginalResponse)),
				)
			} else {
				eventType = sharedevents.RoundTagLookupNotFound
				eventName = "RoundTagLookupNotFound"
				h.logger.InfoContext(ctx, "Tag lookup completed: Tag not found or lookup error",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(responsePayload.UserID)),
					attr.Bool("found_in_payload", responsePayload.Found),
					attr.String("error_in_payload", responsePayload.Error),
					attr.String("original_response", string(responsePayload.OriginalResponse)),
				)
			}

			// Create message using the result payload and determined topic.
			successMsg, err := h.helpers.CreateResultMessage(
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

			// Log the published message details.
			h.logger.InfoContext(ctx, fmt.Sprintf("Publishing %s message", eventName),
				attr.CorrelationIDFromMsg(msg),
				attr.String("message_id", successMsg.UUID),
				attr.String("topic", eventType),
			)
			return []*message.Message{successMsg}, nil
		},
	)

	// Execute the wrapped handler.
	return wrappedHandler(msg)
}

func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetTagByUserIDRequest",
		&leaderboardevents.TagNumberRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagNumberRequestPayload := payload.(*leaderboardevents.TagNumberRequestPayload)

			h.logger.InfoContext(ctx, "Received GetTagByUserIDRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagNumberRequestPayload.UserID)),
				attr.RoundID("round_id", tagNumberRequestPayload.RoundID),
			)

			// Call the service function to get the tag by userID
			result, err := h.leaderboardService.GetTagByUserID(ctx, tagNumberRequestPayload.UserID, tagNumberRequestPayload.RoundID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to get tag by userID",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to get tag by userID: %w", err)
			}

			if result.Failure != nil {
				h.logger.ErrorContext(ctx, "Get tag by userID failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.GetTagNumberFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				responsePayload := result.Success.(*leaderboardevents.GetTagNumberResponsePayload)

				// Determine if tag was found or not
				var eventType string
				if responsePayload.Found {
					h.logger.InfoContext(ctx, "Tag found for user",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
					)
					eventType = leaderboardevents.GetTagNumberResponse
				} else {
					h.logger.InfoContext(ctx, "No tag found for user",
						attr.CorrelationIDFromMsg(msg),
						attr.String("user_id", string(responsePayload.UserID)),
					)
					eventType = leaderboardevents.GetTagByUserIDNotFound
				}

				// Create appropriate response message
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					responsePayload,
					eventType,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Success nor Failure is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from GetTagByUserID",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
