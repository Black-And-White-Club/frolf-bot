package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// mapSourceToServiceUpdateSource converts string sources to ServiceUpdateSource enum
func mapSourceToServiceUpdateSource(source string) sharedtypes.ServiceUpdateSource {
	switch source {
	case "user_creation":
		return sharedtypes.ServiceUpdateSourceCreateUser
	case "manual":
		return sharedtypes.ServiceUpdateSourceManual
	case "round":
		return sharedtypes.ServiceUpdateSourceProcessScores
	case "admin_batch":
		return sharedtypes.ServiceUpdateSourceAdminBatch
	case "tag_swap":
		return sharedtypes.ServiceUpdateSourceTagSwap
	default:
		return sharedtypes.ServiceUpdateSourceManual // Default fallback
	}
}

// HandleTagAssignment handles the TagAssignmentRequested event.
// This can trigger tag assignments or tag swaps depending on the service logic.
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

			// Convert to the unified format
			assignments := []sharedtypes.TagAssignmentRequest{
				{
					UserID:    tagAssignmentRequestedPayload.UserID,
					TagNumber: *tagAssignmentRequestedPayload.TagNumber,
				},
			}

			// Convert string source to ServiceUpdateSource enum
			serviceSource := mapSourceToServiceUpdateSource(tagAssignmentRequestedPayload.Source)

			// Use the public interface method
			result, err := h.leaderboardService.ProcessTagAssignments(
				ctx,
				serviceSource, // Use the proper enum type
				assignments,
				nil, // Individual assignments don't have requesting user
				uuid.UUID(tagAssignmentRequestedPayload.UpdateID),
				uuid.New(), // Generate batch ID for single assignment
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "ProcessTagAssignments failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(tagAssignmentRequestedPayload.UserID)),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process tag assignment: %w", err)
			}

			// Handle failure response
			if result.Failure != nil {
				failureMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardTagAssignmentFailed,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle success response
			if result.Success != nil {
				// Handle tag swap flow
				if swap, ok := result.Success.(*leaderboardevents.TagSwapRequestedPayload); ok {
					h.logger.InfoContext(ctx, "Tag assignment resulted in swap request",
						attr.CorrelationIDFromMsg(msg),
						attr.String("requestor_id", string(swap.RequestorID)),
						attr.String("target_id", string(swap.TargetID)),
					)

					swapMsg, err := h.Helpers.CreateResultMessage(
						msg,
						swap,
						leaderboardevents.TagSwapRequested,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create tag swap message: %w", err)
					}
					return []*message.Message{swapMsg}, nil
				}

				// Handle regular success
				h.logger.InfoContext(ctx, "Tag assignment successful",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(tagAssignmentRequestedPayload.UserID)),
				)

				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardTagAssignmentSuccess,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				return []*message.Message{successMsg}, nil
			}

			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	return wrappedHandler(msg)
}
