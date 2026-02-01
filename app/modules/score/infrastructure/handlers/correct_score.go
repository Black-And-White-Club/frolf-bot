package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleCorrectScoreRequest processes a ScoreUpdateRequest.
func (h *ScoreHandlers) HandleCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	// 1. Execute the service logic
	result, err := h.service.CorrectScore(
		ctx,
		payload.GuildID,
		payload.RoundID,
		payload.UserID,
		payload.Score,
		payload.TagNumber,
	)

	// 2. Handle System Errors (Infrastructure)
	if err != nil {
		return nil, err
	}

	// 3. Handle Business-Level Failures
	// result.Failure is now an 'error' type from the service
	if result.Failure != nil {
		errVal := *result.Failure
		failurePayload := &sharedevents.ScoreUpdateFailedPayloadV1{
			GuildID: payload.GuildID,
			RoundID: payload.RoundID,
			UserID:  payload.UserID,
			Reason:  errVal.Error(),
		}

		return []handlerwrapper.Result{
			{
				Topic:   sharedevents.ScoreUpdateFailedV1,
				Payload: failurePayload,
			},
		}, nil
	}

	// 4. Handle Success Case
	// result.Success is now sharedtypes.ScoreInfo
	if result.Success != nil {
		successPayload := &sharedevents.ScoreUpdatedPayloadV1{
			GuildID: payload.GuildID,
			RoundID: payload.RoundID,
			UserID:  result.Success.UserID,
			Score:   result.Success.Score,
		}

		results := []handlerwrapper.Result{
			{
				Topic:   sharedevents.ScoreUpdatedV1,
				Payload: successPayload,
			},
		}

		// 5. Trigger reprocessing
		scores, err := h.service.GetScoresForRound(ctx, successPayload.GuildID, successPayload.RoundID)
		if err != nil {
			// Infrastructure error during fetch - we still return the success of the update
			return results, nil
		}

		if len(scores) > 0 {
			reprocessPayload := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID: successPayload.GuildID,
				RoundID: successPayload.RoundID,
				Scores:  scores,
			}
			results = append(results, handlerwrapper.Result{
				Topic:   sharedevents.ProcessRoundScoresRequestedV1,
				Payload: reprocessPayload,
			})
		}

		return results, nil
	}

	return nil, errors.New("unexpected result from service: neither success nor failure")
}
