package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
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

			// Call the service to process round scores.
			result, err := h.scoreService.ProcessRoundScores(
				ctx,
				processRoundScoresRequestPayload.GuildID,
				processRoundScoresRequestPayload.RoundID,
				processRoundScoresRequestPayload.Scores,
				processRoundScoresRequestPayload.Overwrite,
			)

			// Handle direct system errors from the service.
			if err != nil && result.Failure == nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					GuildID: processRoundScoresRequestPayload.GuildID,
					RoundID: processRoundScoresRequestPayload.RoundID,
					Error:   err.Error(),
				}
				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message for system error: %w", errCreateResult)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle business-level failures returned by the service via result.Failure.
			if result.Failure != nil {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				failurePayload, ok := result.Failure.(*scoreevents.ProcessRoundScoresFailurePayload)
				if !ok {
					return nil, fmt.Errorf("unexpected failure payload type from service: expected *scoreevents.ProcessRoundScoresFailurePayload, got %T", result.Failure)
				}

				failureMsg, errCreateResult := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ProcessRoundScoresFailure)
				if errCreateResult != nil {
					return nil, fmt.Errorf("failed to create failure message from result failure payload: %w", errCreateResult)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Process success case
			successPayload, ok := result.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
			if !ok {
				h.metrics.RecordRoundScoresProcessingAttempt(ctx, false, processRoundScoresRequestPayload.RoundID)
				return nil, fmt.Errorf("unexpected result from service: expected *scoreevents.ProcessRoundScoresSuccessPayload, got %T", result.Success)
			}

			tagMappings := successPayload.TagMappings

			batchAssignments := make([]sharedevents.TagAssignmentInfo, 0, len(tagMappings))
			for _, tm := range tagMappings {
				batchAssignments = append(batchAssignments, sharedevents.TagAssignmentInfo{
					UserID:    tm.DiscordID,
					TagNumber: tm.TagNumber,
				})
			}

			// FIX: Generate a proper UUID for batch ID instead of concatenating
			batchID := uuid.New().String() // ✅ Generate a proper UUID

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
