package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleProcessRoundScoresRequest handles incoming messages for processing round scores.
// - Singles rounds propagate tag assignments to leaderboard.
// - Team/group rounds only update DB, flow terminates here.
func (h *ScoreHandlers) HandleProcessRoundScoresRequest(
	ctx context.Context,
	payload *sharedevents.ProcessRoundScoresRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	result, err := h.service.ProcessRoundScores(
		ctx,
		payload.GuildID,
		payload.RoundID,
		payload.Scores,
		payload.Overwrite,
	)

	if err != nil && result.Failure == nil {
		return nil, err
	}

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

	// 🔚 Non-singles rounds terminate here (DB updated only)
	if payload.RoundMode != sharedtypes.RoundModeSingles {
		return nil, nil
	}

	successPayload, ok := result.Success.(*sharedevents.ProcessRoundScoresSucceededPayloadV1)
	if !ok {
		return nil, errors.New("unexpected success payload type")
	}

	batchAssignments := make([]sharedevents.TagAssignmentInfoV1, 0, len(successPayload.TagMappings))
	for _, tm := range successPayload.TagMappings {
		batchAssignments = append(batchAssignments, sharedevents.TagAssignmentInfoV1{
			UserID:    tm.DiscordID,
			TagNumber: tm.TagNumber,
		})
	}

	batchPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		RequestingUserID: "score-service",
		BatchID:          uuid.New().String(),
		Assignments:      batchAssignments,
	}

	return []handlerwrapper.Result{
		{
			Topic:   sharedevents.LeaderboardBatchTagAssignmentRequestedV1,
			Payload: batchPayload,
		},
	}, nil
}
