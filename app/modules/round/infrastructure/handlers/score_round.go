package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScoreUpdateRequest handles requests to update a participant's score.
func (h *RoundHandlers) HandleScoreUpdateRequest(
	ctx context.Context,
	payload *roundevents.ScoreUpdateRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ValidateScoreUpdateRequest(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundScoreUpdateValidatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleScoreUpdateValidated processes a validated score update and applies it to the round.
func (h *RoundHandlers) HandleScoreUpdateValidated(
	ctx context.Context,
	payload *roundevents.ScoreUpdateValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.UpdateParticipantScore(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundParticipantScoreUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleScoreBulkUpdateRequest handles bulk score overrides for a round.
func (h *RoundHandlers) HandleScoreBulkUpdateRequest(
	ctx context.Context,
	payload *roundevents.ScoreBulkUpdateRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, sharedtypes.ValidationError{Message: "bulk score update payload is nil"}
	}

	result, err := h.service.UpdateParticipantScoresBulk(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundScoresBulkUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleParticipantScoreUpdated checks if all scores have been submitted after an update.
func (h *RoundHandlers) HandleParticipantScoreUpdated(
	ctx context.Context,
	payload *roundevents.ParticipantScoreUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "HandleParticipantScoreUpdated called",
		attr.String("round_id", payload.RoundID.String()),
		attr.String("user_id", string(payload.UserID)),
	)

	result, err := h.service.CheckAllScoresSubmitted(ctx, *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "CheckAllScoresSubmitted failed", attr.Error(err))
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "all scores submitted check failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundFinalizationFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		if allScoresData, ok := result.Success.(*roundevents.AllScoresSubmittedPayloadV1); ok {
			return []handlerwrapper.Result{
				{Topic: roundevents.RoundAllScoresSubmittedV1, Payload: allScoresData},
			}, nil
		}

		if notAllScoresData, ok := result.Success.(*roundevents.ScoresPartiallySubmittedPayloadV1); ok {
			return []handlerwrapper.Result{
				{Topic: roundevents.RoundScoresPartiallySubmittedV1, Payload: notAllScoresData},
			}, nil
		}

		return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from CheckAllScoresSubmitted"}
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from CheckAllScoresSubmitted service"}
}
