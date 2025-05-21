package scorehandlers

import (
	"context"
	"fmt"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
				// If payload type is incorrect, record metric and return an error to Watermill.
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, sharedtypes.RoundID{})
				return nil, fmt.Errorf("invalid payload type: expected ProcessRoundScoresRequestPayload")
			}

			if processRoundScoresRequestPayload == nil {
				// If payload is nil after type assertion, record metric and return an error.
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, sharedtypes.RoundID{})
				return nil, fmt.Errorf("received nil ProcessRoundScoresRequestPayload after type assertion")
			}

			// Call the service to process round scores.
			// The service will return a ScoreOperationResult that contains either a Success or Failure payload.
			result, err := h.scoreService.ProcessRoundScores(ctx, processRoundScoresRequestPayload.RoundID, processRoundScoresRequestPayload.Scores)

			// Handle direct system errors from the service.
			// If `err` is not nil AND `result.Failure` is nil, it means a deeper, unhandled system error occurred.
			if err != nil && result.Failure == nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				// Create a failure payload for the system error and publish it.
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   err.Error(),
				}
				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message for system error: %w", errCreateResult)
				}
				// Return nil, nil to Watermill to acknowledge the message, as we've published a failure response.
				return []*message.Message{failureMsg}, nil
			}

			// Handle business-level failures returned by the service via result.Failure.
			if result.Failure != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				// Assert that result.Failure is the expected ProcessRoundScoresFailurePayload.
				failurePayload, ok := result.Failure.(*scoreevents.ProcessRoundScoresFailurePayload)
				if !ok {
					// If the failure payload type is unexpected, log and return an internal error.
					return nil, fmt.Errorf("unexpected failure payload type from service: expected *scoreevents.ProcessRoundScoresFailurePayload, got %T", result.Failure)
				}

				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message from result failure payload: %w", errCreateResult)
				}
				// Return nil, nil to Watermill to acknowledge the message.
				return []*message.Message{failureMsg}, nil
			}

			// If no direct error and no business failure, then it must be a success.
			// Assert that result.Success is the expected *scoreevents.ProcessRoundScoresSuccessPayload.
			successPayload, ok := result.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
			if !ok {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unexpected result from service: expected *scoreevents.ProcessRoundScoresSuccessPayload, got %T", result.Success)
			}

			tagMappings := successPayload.TagMappings // Extract TagMappings from the payload

			batchAssignments := make([]sharedevents.TagAssignmentInfo, 0, len(tagMappings))
			for _, tm := range tagMappings {
				batchAssignments = append(batchAssignments, sharedevents.TagAssignmentInfo{
					UserID:    tm.DiscordID,
					TagNumber: tm.TagNumber,
				})
			}

			batchID := fmt.Sprintf("%s-%d", processRoundScoresRequestPayload.RoundID, time.Now().UnixNano())
			batchPayload := &sharedevents.BatchTagAssignmentRequestedPayload{
				RequestingUserID: "score-service",
				BatchID:          batchID,
				Assignments:      batchAssignments,
			}

			batchAssignMsg, err := h.Helpers.CreateResultMessage(
				msg,
				batchPayload,
				sharedevents.LeaderboardBatchTagAssignmentRequested,
			)
			if err != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("failed to create batch tag assignment message: %w", err)
			}

			h.metrics.RecordRoundScoresProcessingAttempt(ctx, true, processRoundScoresRequestPayload.RoundID)
			return []*message.Message{batchAssignMsg}, nil
		},
	)(msg)
}
