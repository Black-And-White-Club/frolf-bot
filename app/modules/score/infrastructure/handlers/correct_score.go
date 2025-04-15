package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCorrectScoreRequest handles correct score requests.
func (h *ScoreHandlers) HandleCorrectScoreRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleCorrectScoreRequest",
		&scoreevents.ScoreUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateRequestPayload, ok := payload.(*scoreevents.ScoreUpdateRequestPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type: expected ScoreUpdateRequestPayload")
			}

			// Log received event
			h.logger.InfoContext(ctx, "Received CorrectScoreRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", scoreUpdateRequestPayload.RoundID),
				attr.String("user_id", string(scoreUpdateRequestPayload.UserID)),
				attr.Int("score", int(scoreUpdateRequestPayload.Score)),
				attr.Any("tag_number", scoreUpdateRequestPayload.TagNumber),
			)

			// Call the service
			result, err := h.scoreService.CorrectScore(ctx, scoreUpdateRequestPayload.RoundID, scoreUpdateRequestPayload.UserID, scoreUpdateRequestPayload.Score, scoreUpdateRequestPayload.TagNumber)
			// Check if an error occurred
			if err != nil {
				// Handle the error
				h.logger.ErrorContext(ctx, "Error correcting score",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, err
			}

			// Determine if success or failure
			var eventType string
			var payloadToPublish interface{}
			if result.Failure != nil {
				eventType = scoreevents.ScoreUpdateFailure
				payloadToPublish = result.Failure
			} else if result.Success != nil {
				eventType = scoreevents.ScoreUpdateSuccess
				payloadToPublish = result.Success
			} else {
				return nil, fmt.Errorf("unexpected result from service")
			}

			// Create and return message
			responseMsg, err := h.helpers.CreateResultMessage(msg, payloadToPublish, eventType)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create response message",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create response message: %w", err)
			}

			return []*message.Message{responseMsg}, nil
		},
	)(msg)
}
