package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *ScoreHandlers) HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScoreUpdateRequest",
		&scoreevents.ScoreUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateRequestPayload := payload.(*scoreevents.ScoreUpdateRequestPayload)

			h.logger.Info("Received ScoreUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.Int64("round_id", int64(scoreUpdateRequestPayload.RoundID)),
				attr.String("user_id", string(scoreUpdateRequestPayload.UserID)),
				attr.Int("score", int(scoreUpdateRequestPayload.Score)),
				attr.Int("tag_number", int(*scoreUpdateRequestPayload.TagNumber)),
			)

			// Call the service function to correct the score
			result, err := h.scoreService.CorrectScore(ctx, msg, *scoreUpdateRequestPayload)
			if err != nil {
				h.logger.Error("Failed to correct score",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)

				failureResultPayload := &scoreevents.ScoreUpdateResponsePayload{
					Success: false,
					RoundID: scoreUpdateRequestPayload.RoundID,
					Error:   err.Error(),
				}

				failureMsg, errCreateResult := h.helpers.CreateResultMessage(
					msg,
					failureResultPayload,
					scoreevents.ScoreUpdateFailure,
				)
				if errCreateResult != nil {
					h.logger.Error("Failed to create failure message",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errCreateResult),
					)
					return nil, fmt.Errorf("failed to correct score: %w", err)
				}

				h.logger.Info("ScoreUpdateRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordHandlerFailure("HandleScoreUpdateRequest")

				return []*message.Message{failureMsg}, nil
			}

			// Check if the operation was successful
			if result.Success != nil {
				score, ok := result.Success.(*scoredb.Score)
				if !ok {
					h.logger.Error("Failed to convert result to Score",
						attr.CorrelationIDFromMsg(msg),
					)
					h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
					return nil, fmt.Errorf("failed to convert result to Score")
				}

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					&scoreevents.ScoreUpdateResponsePayload{
						Success: true,
						RoundID: scoreUpdateRequestPayload.RoundID,
						Participant: scoreevents.Participant{
							UserID:    score.UserID,
							TagNumber: score.TagNumber,
							Score:     score.Score,
						},
					},
					scoreevents.ScoreUpdateSuccess,
				)
				if err != nil {
					h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				h.logger.Info("ScoreUpdateRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordScoreUpdateAttempt(true, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
				h.metrics.RecordHandlerSuccess("HandleScoreUpdateRequest")

				return []*message.Message{successMsg}, nil
			} else if result.Failure != nil {
				failure, ok := result.Failure.(error)
				if !ok {
					h.logger.Error("Failed to convert result to error",
						attr.CorrelationIDFromMsg(msg),
					)
					h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
					return nil, fmt.Errorf("failed to convert result to error")
				}

				// Create failure message to publish
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					&scoreevents.ScoreUpdateResponsePayload{
						Success: false,
						RoundID: scoreUpdateRequestPayload.RoundID,
						Error:   failure.Error(),
					},
					scoreevents.ScoreUpdateFailure,
				)
				if err != nil {
					h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}

				h.logger.Info("ScoreUpdateRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
				h.metrics.RecordHandlerFailure("HandleScoreUpdateRequest")

				return []*message.Message{failureMsg}, nil
			} else {
				// Handle unknown result
				h.logger.Error("Unknown result from CorrectScore",
					attr.CorrelationIDFromMsg(msg),
				)
				h.metrics.RecordScoreUpdateAttempt(false, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID)
				return nil, fmt.Errorf("unknown result from CorrectScore")
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
