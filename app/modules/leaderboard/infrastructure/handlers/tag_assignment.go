package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
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
				attr.String("source", tagAssignmentRequestedPayload.Source),
			)

			result, err := h.leaderboardService.TagAssignmentRequested(ctx, *tagAssignmentRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "TagAssignmentRequested failed", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, err
			}

			if result == (leaderboardservice.LeaderboardOperationResult{}) {
				h.logger.ErrorContext(ctx, "Service returned empty result", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("empty result from service")
			}

			var outMsgs []*message.Message

			if result.Failure != nil {
				// Always publish failure to leaderboard stream
				failureMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardTagAssignmentFailed,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				outMsgs = append(outMsgs, failureMsg)
				return outMsgs, nil
			}

			if result.Success != nil {
				// Handle tag swap flow
				if swap, ok := result.Success.(*leaderboardevents.TagSwapRequestedPayload); ok {
					swapMsg, err := h.Helpers.CreateResultMessage(
						msg,
						swap,
						leaderboardevents.TagSwapRequested,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create tag swap message: %w", err)
					}
					outMsgs = append(outMsgs, swapMsg)
					return outMsgs, nil
				}

				// Route success based on Source field
				switch tagAssignmentRequestedPayload.Source {
				case "user_creation":
					userMsg, err := h.Helpers.CreateResultMessage(
						msg,
						result.Success,
						leaderboardevents.LeaderboardTagAssignmentSuccess,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create user creation message: %w", err)
					}
					outMsgs = append(outMsgs, userMsg)
				default:
					successMsg, err := h.Helpers.CreateResultMessage(
						msg,
						result.Success,
						leaderboardevents.LeaderboardTagAssignmentSuccess,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create success message: %w", err)
					}
					outMsgs = append(outMsgs, successMsg)
				}
				return outMsgs, nil
			}

			h.logger.ErrorContext(ctx, "Service returned result with neither Success nor Failure payload set, and no error",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected service result: neither success nor failure")
		},
	)

	return wrappedHandler(msg)
}
