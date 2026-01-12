package leaderboardhandlers

import (
	"context"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagSwapRequested handles the TagSwapRequested event.
func (h *LeaderboardHandlers) HandleTagSwapRequested(
	ctx context.Context,
	payload *leaderboardevents.TagSwapRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received TagSwapRequested event",
		attr.ExtractCorrelationID(ctx),
		attr.String("requestor_id", string(payload.RequestorID)),
		attr.String("target_id", string(payload.TargetID)),
	)

	// Call the service function to handle the event
	result, err := h.leaderboardService.TagSwapRequested(ctx, payload.GuildID, *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to handle TagSwapRequested event",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Failure != nil {
		failedPayload, ok := result.Failure.(*leaderboardevents.TagSwapFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}
		h.logger.InfoContext(ctx, "Tag swap failed",
			attr.ExtractCorrelationID(ctx),
			attr.Any("failure_payload", failedPayload),
		)

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.TagSwapFailedV1, Payload: failedPayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*leaderboardevents.TagSwapProcessedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}
		h.logger.InfoContext(ctx, "Tag swap successful",
			attr.ExtractCorrelationID(ctx),
		)

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.TagSwapProcessedV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("tag swap service returned unexpected result")
}
