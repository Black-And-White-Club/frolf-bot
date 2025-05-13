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
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, sharedtypes.RoundID{})
				return nil, fmt.Errorf("invalid payload type: expected ProcessRoundScoresRequestPayload")
			}

			if processRoundScoresRequestPayload == nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, sharedtypes.RoundID{})
				return nil, fmt.Errorf("received nil ProcessRoundScoresRequestPayload after type assertion")
			}

			result, err := h.scoreService.ProcessRoundScores(ctx, processRoundScoresRequestPayload.RoundID, processRoundScoresRequestPayload.Scores)
			if err != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   err.Error(),
				}
				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errCreateResult)
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Error != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   result.Error.Error(),
				}
				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message from result error: %w", errCreateResult)
				}
				return []*message.Message{failureMsg}, nil
			}

			tagMappings, ok := result.Success.([]sharedtypes.TagMapping)
			if !ok {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unexpected result type from service: expected []sharedtypes.TagMapping, got %T", result.Success)
			}

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
