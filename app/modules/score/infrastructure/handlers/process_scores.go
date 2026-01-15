package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleProcessRoundScoresRequest handles the incoming message for processing round scores.
func (h *ScoreHandlers) HandleProcessRoundScoresRequest(ctx context.Context, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	// Call the service to process round scores.
	result, err := h.scoreService.ProcessRoundScores(
		ctx,
		payload.GuildID,
		payload.RoundID,
		payload.Scores,
		payload.Overwrite,
	)

	// Handle system errors from the service.
	if err != nil && result.Failure == nil {
		return nil, err
	}

	// Handle business-level failures returned by the service via result.Failure.
	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*sharedevents.ProcessRoundScoresFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected failure payload type from service")
		}

		return []handlerwrapper.Result{
			{
				Topic:   sharedevents.ProcessRoundScoresFailedV1,
				Payload: failurePayload,
			},
		}, nil
	}

	// Process success case
	successPayload, ok := result.Success.(*sharedevents.ProcessRoundScoresSucceededPayloadV1)
	if !ok {
		return nil, errors.New("unexpected result from service: expected ProcessRoundScoresSucceededPayloadV1")
	}

	tagMappings := successPayload.TagMappings

	batchAssignments := make([]sharedevents.TagAssignmentInfoV1, 0, len(tagMappings))
	for _, tm := range tagMappings {
		batchAssignments = append(batchAssignments, sharedevents.TagAssignmentInfoV1{
			UserID:    tm.DiscordID,
			TagNumber: tm.TagNumber,
		})
	}

	batchID := uuid.New().String()

	batchPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		RequestingUserID: "score-service",
		BatchID:          batchID,
		Assignments:      batchAssignments,
	}

	return []handlerwrapper.Result{
		{
			Topic:   sharedevents.LeaderboardBatchTagAssignmentRequestedV1,
			Payload: batchPayload,
		},
	}, nil
}
