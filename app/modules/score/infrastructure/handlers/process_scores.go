package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleProcessRoundScoresRequest handles the incoming message for processing round scores.
func (h *ScoreHandlers) HandleProcessRoundScoresRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleProcessRoundScoresRequest",
		&scoreevents.ProcessRoundScoresRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			processRoundScoresRequestPayload, ok := payload.(*scoreevents.ProcessRoundScoresRequestPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type: expected ProcessRoundScoresRequestPayload")
			}

			// Call the service function
			result, err := h.scoreService.ProcessRoundScores(ctx, processRoundScoresRequestPayload.RoundID, processRoundScoresRequestPayload.Scores)
			if err != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)

				// Create failure event
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   err.Error(),
				}

				// Create failure message
				failureMsg, errCreateResult := h.helpers.CreateResultMessage(
					msg,
					failurePayload,
					scoreevents.ProcessRoundScoresFailure,
				)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errCreateResult)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success == nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unknown result from ProcessRoundScores")
			}

			successPayload, ok := result.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
			if !ok {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unexpected result type: %T", result.Success)
			}

			// Create success message
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				successPayload,
				scoreevents.ProcessRoundScoresSuccess,
			)
			if err != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			h.metrics.RecordRoundScoresProcessingAttempt(ctx, true, processRoundScoresRequestPayload.RoundID)
			return []*message.Message{successMsg}, nil
		},
	)(msg)
}
