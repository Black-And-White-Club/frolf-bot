package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCorrectScoreRequest processes a ScoreUpdateRequest message.
// It calls the ScoreService to correct a score and publishes either a success or failure message.
// If the service returns a business-level failure (e.g., score record not found),
// it publishes a ScoreUpdateFailure message and acknowledges the original message.
// If a deeper system error occurs that the service couldn't handle, it returns an error to Watermill.
func (h *ScoreHandlers) HandleCorrectScoreRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleCorrectScoreRequest",
		&scoreevents.ScoreUpdateRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateRequestPayload, ok := payload.(*scoreevents.ScoreUpdateRequestedPayloadV1)
			if !ok {
				return nil, fmt.Errorf("invalid payload type: expected ScoreUpdateRequestedPayloadV1")
			}

			result, err := h.scoreService.CorrectScore(
				ctx,
				sharedtypes.GuildID(scoreUpdateRequestPayload.GuildID),
				scoreUpdateRequestPayload.RoundID,
				scoreUpdateRequestPayload.UserID,
				scoreUpdateRequestPayload.Score,
				scoreUpdateRequestPayload.TagNumber,
			)

			// Check if a fundamental error occurred that the service couldn't handle, AND
			// the service did not already provide a business-level failure payload.
			// If such an error exists, return it to Watermill for potential retry.
			if err != nil && result.Failure == nil {
				return nil, err
			}

			// If the service returned a business-level failure payload,
			// create and publish a ScoreUpdateFailure message.
			if result.Failure != nil {
				failurePayload, ok := result.Failure.(*scoreevents.ScoreUpdateFailedPayloadV1)
				if !ok {
					// If the failure payload type is unexpected, return an internal error.
					return nil, fmt.Errorf("unexpected failure payload type from service: expected ScoreUpdateFailedPayloadV1, got %T", result.Failure)
				}
				// Create the failure message.
				failureMsg, err := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ScoreUpdateFailedV1)
				if err != nil {
					return nil, fmt.Errorf("failed to create ScoreUpdateFailure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			// If the service returned a success payload,
			// create and publish a ScoreUpdateSuccess message.
			if result.Success != nil {
				successPayload, ok := result.Success.(*scoreevents.ScoreUpdatedPayloadV1)
				if !ok {
					// If the success payload type is unexpected, return an internal error.
					return nil, fmt.Errorf("unexpected success payload type from service: expected *scoreevents.ScoreUpdatedPayloadV1, got %T", result.Success)
				}

				// Create the success message.
				successMsg, err := h.Helpers.CreateResultMessage(msg, successPayload, scoreevents.ScoreUpdatedV1)
				if err != nil {
					return nil, fmt.Errorf("failed to create ScoreUpdateSuccess message: %w", err)
				}

				// Trigger reprocessing by getting the round scores and publishing a reprocess request
				scores, err := h.scoreService.GetScoresForRound(ctx, successPayload.GuildID, successPayload.RoundID)
				if err != nil {
					// Log the error but don't fail the whole operation - the score update succeeded
					h.logger.WarnContext(ctx, "Failed to get scores for reprocessing after score update",
						"round_id", successPayload.RoundID,
						"error", err,
					)
					return []*message.Message{successMsg}, nil
				}

				if len(scores) > 0 {
					// Create a reprocess request message
					reprocessPayload := scoreevents.ProcessRoundScoresRequestedPayloadV1{
						GuildID: successPayload.GuildID,
						RoundID: successPayload.RoundID,
						Scores:  scores,
					}
					reprocessMsg, err := h.Helpers.CreateResultMessage(msg, &reprocessPayload, scoreevents.ProcessRoundScoresRequestedV1)
					if err != nil {
						// Log the error but don't fail the whole operation
						h.logger.WarnContext(ctx, "Failed to create reprocess message after score update",
							"round_id", successPayload.RoundID,
							"error", err,
						)
						return []*message.Message{successMsg}, nil
					}
					// Return both the success message and the reprocess request
					return []*message.Message{successMsg, reprocessMsg}, nil
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither success nor failure payload was returned, it's an unexpected state.
			return nil, fmt.Errorf("unexpected result from service: neither success nor failure")
		},
	)(msg)
}
