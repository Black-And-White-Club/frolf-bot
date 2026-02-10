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

	// 1. Handle Hard System/Infra Errors
	if err != nil {
		return nil, err
	}

	// 2. Handle Domain Failures (Dereferencing the *error)
	if result.Failure != nil {
		errVal := *result.Failure // Resolve the pointer to interface

		return []handlerwrapper.Result{
			{
				Topic: sharedevents.ProcessRoundScoresFailedV1,
				Payload: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Reason:  errVal.Error(),
				},
			},
		}, nil
	}

	//  Non-singles rounds terminate here (DB updated only)
	if payload.RoundMode != sharedtypes.RoundModeSingles {
		return nil, nil
	}

	// 3. Handle Success Case
	if result.Success != nil {
		// Use the tag mappings returned in our custom Success struct
		batchAssignments := make([]sharedevents.TagAssignmentInfoV1, 0, len(result.Success.TagMappings))
		for _, tm := range result.Success.TagMappings {
			batchAssignments = append(batchAssignments, sharedevents.TagAssignmentInfoV1{
				UserID:    tm.DiscordID,
				TagNumber: tm.TagNumber,
			})
		}

		if len(batchAssignments) == 0 {
			return nil, nil
		}

		batchPayload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
			ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
			RequestingUserID: "score-service",
			BatchID:          uuid.New().String(),
			Assignments:      batchAssignments,
			RoundID:          &payload.RoundID,
			Source:           sharedtypes.ServiceUpdateSourceProcessScores,
		}

		return []handlerwrapper.Result{
			{
				Topic:   sharedevents.LeaderboardBatchTagAssignmentRequestedV1,
				Payload: batchPayload,
			},
		}, nil
	}

	return nil, errors.New("unexpected result from service: neither success nor failure")
}
