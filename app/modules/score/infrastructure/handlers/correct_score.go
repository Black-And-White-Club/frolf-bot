package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleCorrectScoreRequest processes a ScoreUpdateRequest.
// It calls the ScoreService to correct a score and returns either a success or failure event.
func (h *ScoreHandlers) HandleCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	// 1. Execute the service logic
	result, err := h.scoreService.CorrectScore(
		ctx,
		payload.GuildID,
		payload.RoundID,
		payload.UserID,
		payload.Score,
		payload.TagNumber,
	)

	// 2. Handle System Errors (e.g. DB connection issues)
	// We return the error so the message can be retried by the infrastructure if needed.
	if err != nil {
		return nil, err
	}

	// 3. Handle Business-Level Failures (Handled by the service)
	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*sharedevents.ScoreUpdateFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected failure payload type from service")
		}

		return []handlerwrapper.Result{
			{
				Topic:   sharedevents.ScoreUpdateFailedV1,
				Payload: failurePayload,
			},
		}, nil
	}

	// 4. Handle Success Case
	if result.Success != nil {
		successPayload, ok := result.Success.(*sharedevents.ScoreUpdatedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected success payload type from service")
		}

		results := []handlerwrapper.Result{
			{
				Topic:   sharedevents.ScoreUpdatedV1,
				Payload: successPayload,
			},
		}

		// 5. Trigger reprocessing
		// Fetch scores to include in the reprocess request
		scores, err := h.scoreService.GetScoresForRound(ctx, successPayload.GuildID, successPayload.RoundID)
		if err != nil {
			// We log a warning but return the success event.
			// Do NOT return an error here, or the handler will retry and double-publish.
			h.logger.WarnContext(ctx, "Score updated but failed to fetch scores for reprocessing",
				"round_id", successPayload.RoundID,
				"error", err,
			)
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

	// 6. Final Fallback - Matches Unit Test expectation exactly
	return nil, errors.New("unexpected result from service: neither success nor failure")
}
