package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *ScoreHandlers) HandleCorrectScoreRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleCorrectScoreRequest",
		&scoreevents.ScoreUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateRequestPayload, ok := payload.(*scoreevents.ScoreUpdateRequestPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type: expected ScoreUpdateRequestPayload")
			}

			result, err := h.scoreService.CorrectScore(
				ctx,
				scoreUpdateRequestPayload.RoundID,
				scoreUpdateRequestPayload.UserID,
				scoreUpdateRequestPayload.Score,
				scoreUpdateRequestPayload.TagNumber,
			)
			if err != nil {
				return nil, err
			}

			if result.Failure != nil {
				failurePayload, ok := result.Failure.(*scoreevents.ScoreUpdateFailurePayload)
				if !ok {
					return nil, fmt.Errorf("unexpected failure payload type from service: expected ScoreUpdateFailurePayload, got %T", result.Failure)
				}
				failureMsg, err := h.Helpers.CreateResultMessage(msg, failurePayload, scoreevents.ScoreUpdateFailure)
				if err != nil {
					return nil, fmt.Errorf("failed to create ScoreUpdateFailure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*scoreevents.ScoreUpdateSuccessPayload)
				if !ok {
					return nil, fmt.Errorf("unexpected success payload type from service: expected *scoreevents.ScoreUpdateSuccessPayload, got %T", result.Success)
				}

				successMsg, err := h.Helpers.CreateResultMessage(msg, successPayload, scoreevents.ScoreUpdateSuccess)
				if err != nil {
					return nil, fmt.Errorf("failed to create ScoreUpdateSuccess message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			return nil, fmt.Errorf("unexpected result from service: neither success nor failure")
		},
	)(msg)
}
