package leaderboardhandlers

import (
	"context"
	"errors"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received BatchTagAssignmentRequested event",
		attr.ExtractCorrelationID(ctx),
		attr.String("batch_id", payload.BatchID),
		attr.String("requesting_user", string(payload.RequestingUserID)),
		attr.Int("assignment_count", len(payload.Assignments)),
	)

	// Convert assignments to the expected format
	assignments := make([]sharedtypes.TagAssignmentRequest, len(payload.Assignments))
	for i, assignment := range payload.Assignments {
		assignments[i] = sharedtypes.TagAssignmentRequest{
			UserID:    assignment.UserID,
			TagNumber: assignment.TagNumber,
		}
	}

	batchID, err := uuid.Parse(payload.BatchID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Invalid batch ID format", attr.Error(err))
		return nil, err
	}

	// Call service with payload as source context
	result, err := h.leaderboardService.ProcessTagAssignments(
		ctx,
		payload.GuildID,
		payload, // Pass the entire payload for source determination
		assignments,
		&payload.RequestingUserID,
		uuid.New(),
		batchID,
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service failed to handle batch assignment", attr.Error(err))
		return nil, err
	}

	// Handle failure response
	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardBatchTagAssignmentFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}
		return []handlerwrapper.Result{
			{Topic: leaderboardevents.LeaderboardBatchTagAssignmentFailedV1, Payload: failurePayload},
		}, nil
	}

	// Handle success response
	if result.Success != nil {
		successPayload, ok := result.Success.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}

		h.logger.InfoContext(ctx, "Batch tag assignment successful",
			attr.ExtractCorrelationID(ctx),
			attr.String("batch_id", payload.BatchID),
		)

		results := []handlerwrapper.Result{
			{Topic: leaderboardevents.LeaderboardBatchTagAssignedV1, Payload: successPayload},
		}

		// Publish scheduled round updates for batch assignments
		changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
		for _, assignment := range assignments {
			changedTags[assignment.UserID] = assignment.TagNumber
		}

		tagUpdatePayload := &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
			GuildID:     payload.GuildID,
			ChangedTags: changedTags,
			UpdatedAt:   time.Now().UTC(),
			Source:      "batch_assignment",
		}

		h.logger.InfoContext(ctx, "Publishing tag updates to scheduled rounds",
			attr.ExtractCorrelationID(ctx),
			attr.String("batch_id", payload.BatchID),
			attr.Int("changed_tags", len(assignments)),
		)

		results = append(results, handlerwrapper.Result{
			Topic:   sharedevents.TagUpdateForScheduledRoundsV1,
			Payload: tagUpdatePayload,
		})

		return results, nil
	}

	return nil, errors.New("batch tag assignment service returned unexpected result")
}
