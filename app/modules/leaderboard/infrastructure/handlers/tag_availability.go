package leaderboardhandlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleTagAvailabilityCheckRequested handles the TagAvailabilityCheckRequested event.
// This handler generates cross-module events to the User module based on tag availability.
func (h *LeaderboardHandlers) HandleTagAvailabilityCheckRequested(
	ctx context.Context,
	payload *leaderboardevents.TagAvailabilityCheckRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received TagAvailabilityCheckRequested event",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.Int("tag_number", int(*payload.TagNumber)),
	)

	result, failure, err := h.leaderboardService.CheckTagAvailability(ctx, payload.GuildID, *payload)

	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to handle TagAvailabilityCheckRequested event",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	if failure != nil {
		failure.GuildID = payload.GuildID // Ensure guild_id is set
		h.logger.InfoContext(ctx, "Tag availability check failed",
			attr.ExtractCorrelationID(ctx),
			attr.Any("failure_payload", failure),
		)
		return []handlerwrapper.Result{
			{Topic: leaderboardevents.TagAvailabilityCheckFailedV1, Payload: failure},
		}, nil
	}

	h.logger.InfoContext(ctx, "Tag availability check successful", attr.ExtractCorrelationID(ctx))

	// Create outcome messages based on tag availability
	if result.Available {
		h.logger.InfoContext(ctx, "Tag is available",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(result.UserID)),
			attr.Int("tag_number", int(*result.TagNumber)),
		)

		// Event 1: Notify User module that tag is available
		availablePayload := &userevents.TagAvailablePayloadV1{
			GuildID:   result.GuildID,
			UserID:    result.UserID,
			TagNumber: *result.TagNumber,
		}

		// Event 2: Request tag assignment for the user
		assignTagPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
			ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
			RequestingUserID: "system",
			BatchID:          uuid.New().String(),
			Assignments: []sharedevents.TagAssignmentInfoV1{
				{
					UserID:    result.UserID,
					TagNumber: *result.TagNumber,
				},
			},
		}

		return []handlerwrapper.Result{
			{Topic: userevents.TagAvailableV1, Payload: availablePayload},
			{Topic: sharedevents.LeaderboardBatchTagAssignmentRequestedV1, Payload: assignTagPayload},
		}, nil
	}

	// Tag is unavailable
	tagUnavailable := &userevents.TagUnavailablePayloadV1{
		GuildID:   result.GuildID,
		UserID:    result.UserID,
		TagNumber: *result.TagNumber,
		Reason:    result.Reason,
	}

	h.logger.InfoContext(ctx, "Tag is not available",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(result.UserID)),
		attr.Int("tag_number", int(*result.TagNumber)),
		attr.String("reason", result.Reason),
	)

	return []handlerwrapper.Result{
		{Topic: userevents.TagUnavailableV1, Payload: tagUnavailable},
	}, nil
}
