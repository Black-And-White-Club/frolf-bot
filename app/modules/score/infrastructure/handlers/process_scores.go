package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *ScoreHandlers) HandleProcessRoundScoresRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleProcessRoundScoresRequest",
		&scoreevents.ProcessRoundScoresRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			processRoundScoresRequestPayload := payload.(*scoreevents.ProcessRoundScoresRequestPayload)

			h.logger.Info("Received ProcessRoundScoresRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.Int64("round_id", int64(processRoundScoresRequestPayload.RoundID)),
			)

			// Call the service function to process the round scores
			result, err := h.scoreService.ProcessRoundScores(ctx, msg, *processRoundScoresRequestPayload)
			if err != nil {
				h.logger.Error("Failed to process round scores",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)

				failureResultPayload := &scoreevents.ProcessRoundScoresResponsePayload{
					Success: false,
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   err.Error(),
				}

				failureMsg, errCreateResult := h.helpers.CreateResultMessage(
					msg,
					failureResultPayload,
					scoreevents.ProcessRoundScoresFailure,
				)
				if errCreateResult != nil {
					h.logger.Error("Failed to create failure message",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errCreateResult),
					)
					return nil, fmt.Errorf("failed to process round scores: %w", err)
				}

				h.logger.Info("ProcessRoundScoresRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordHandlerFailure("HandleProcessRoundScoresRequest")

				return []*message.Message{failureMsg}, nil
			}

			// Check if the operation was successful
			if result.Success != nil {
				scores, ok := result.Success.([]scoreevents.ParticipantScore)
				if !ok {
					h.logger.Error("Failed to convert result to ParticipantScore slice",
						attr.CorrelationIDFromMsg(msg),
					)
					h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
					return nil, fmt.Errorf("failed to convert result to ParticipantScore slice")
				}

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					&scoreevents.ProcessRoundScoresResponsePayload{
						Success: true,
						RoundID: processRoundScoresRequestPayload.RoundID,
						Scores:  scores,
					},
					scoreevents.ProcessRoundScoresSuccess,
				)
				if err != nil {
					h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				h.logger.Info("ProcessRoundScoresRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordRoundScoresProcessingAttempt(true, processRoundScoresRequestPayload.RoundID)
				h.metrics.RecordHandlerSuccess("HandleProcessRoundScoresRequest")

				return []*message.Message{successMsg}, nil
			} else if result.Failure != nil {
				failure, ok := result.Failure.(error)
				if !ok {
					h.logger.Error("Failed to convert result to error",
						attr.CorrelationIDFromMsg(msg),
					)
					h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
					return nil, fmt.Errorf("failed to convert result to error")
				}

				// Create failure message to publish
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					&scoreevents.ProcessRoundScoresResponsePayload{
						Success: false,
						RoundID: processRoundScoresRequestPayload.RoundID,
						Error:   failure.Error(),
					},
					scoreevents.ProcessRoundScoresFailure,
				)
				if err != nil {
					h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}

				h.logger.Info("ProcessRoundScoresRequest event processed", attr.CorrelationIDFromMsg(msg))
				h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
				h.metrics.RecordHandlerFailure("HandleProcessRoundScoresRequest")

				return []*message.Message{failureMsg}, nil
			} else {
				// Handle unknown result
				h.logger.Error("Unknown result from ProcessRoundScores",
					attr.CorrelationIDFromMsg(msg),
				)
				h.metrics.RecordRoundScoresProcessingAttempt(false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unknown result from ProcessRoundScores")
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
