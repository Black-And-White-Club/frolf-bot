package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetLeaderboardRequest",
		&leaderboardevents.GetLeaderboardRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.logger.Info("Received GetLeaderboardRequest event",
				attr.CorrelationIDFromMsg(msg),
			)

			// Call the service function to get the leaderboard
			result, err := h.leaderboardService.GetLeaderboard(ctx)
			if err != nil {
				h.logger.Error("Failed to get leaderboard",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to get leaderboard: %w", err)
			}

			if result.Failure != nil {
				h.logger.Error("Get leaderboard failed",
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
				h.logger.Info("Get leaderboard successful",
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
			h.logger.Error("Unexpected result from GetLeaderboard",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleGetTagByUserIDRequest handles the GetTagByUserIDRequest event.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetTagByUserIDRequest",
		&leaderboardevents.TagNumberRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagNumberRequestPayload := payload.(*leaderboardevents.TagNumberRequestPayload)

			h.logger.Info("Received GetTagByUserIDRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagNumberRequestPayload.UserID)),
				attr.RoundID("round_id", tagNumberRequestPayload.RoundID),
			)

			// Call the service function to get the tag by userID
			result, err := h.leaderboardService.GetTagByUserID(ctx, tagNumberRequestPayload.UserID, tagNumberRequestPayload.RoundID)
			if err != nil {
				h.logger.Error("Failed to get tag by userID",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to get tag by userID: %w", err)
			}

			if result.Failure != nil {
				h.logger.Error("Get tag by userID failed",
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
				h.logger.Info("Get tag by userID successful",
					attr.CorrelationIDFromMsg(msg),
				)

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.GetTagNumberResponse,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Success nor Failure is set, return an error
			h.logger.Error("Unexpected result from GetTagByUserID",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
