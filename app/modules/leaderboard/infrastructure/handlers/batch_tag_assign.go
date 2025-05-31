package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleBatchTagAssignmentRequested handles the BatchTagAssignmentRequested event.
// This is for admin/manual batch operations outside of rounds - multiple tags assigned at once.
func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleBatchTagAssignmentRequested",
		&sharedevents.BatchTagAssignmentRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			batchPayload := payload.(*sharedevents.BatchTagAssignmentRequestedPayload)

			h.logger.InfoContext(ctx, "Received BatchTagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("batch_id", batchPayload.BatchID),
				attr.String("requesting_user", string(batchPayload.RequestingUserID)),
				attr.Int("assignment_count", len(batchPayload.Assignments)),
			)

			// Convert assignments to the expected format
			assignments := make([]sharedtypes.TagAssignmentRequest, len(batchPayload.Assignments))
			for i, assignment := range batchPayload.Assignments {
				assignments[i] = sharedtypes.TagAssignmentRequest{
					UserID:    assignment.UserID,
					TagNumber: assignment.TagNumber,
				}
			}

			// Parse batch ID
			batchID, err := uuid.Parse(batchPayload.BatchID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Invalid batch ID format",
					attr.CorrelationIDFromMsg(msg),
					attr.String("batch_id", batchPayload.BatchID),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("invalid batch ID format: %w", err)
			}

			// Use the public interface method
			result, err := h.leaderboardService.ProcessTagAssignments(
				ctx,
				sharedtypes.ServiceUpdateSourceAdminBatch,
				assignments,
				&batchPayload.RequestingUserID,
				uuid.New(), // Generate operation ID
				batchID,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Service failed to handle batch assignment",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to process batch tag assignments: %w", err)
			}

			// Handle failure response
			if result.Failure != nil {
				failureMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed, // Fixed: Use correct event constant
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle success response
			if result.Success != nil {
				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardBatchTagAssigned, // Fixed: Use correct event constant
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
