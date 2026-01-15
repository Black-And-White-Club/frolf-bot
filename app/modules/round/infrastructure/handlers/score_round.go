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
	result, err := h.roundService.ValidateScoreUpdateRequest(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "score update validation failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScoreUpdateErrorV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScoreUpdateValidatedV1, Payload: result.Success},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from ValidateScoreUpdateRequest service"}
}

// HandleScoreUpdateValidated processes a validated score update and applies it to the round.
func (h *RoundHandlers) HandleScoreUpdateValidated(
	ctx context.Context,
	payload *roundevents.ScoreUpdateValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.roundService.UpdateParticipantScore(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "participant score update failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScoreUpdateErrorV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantScoreUpdatedV1, Payload: result.Success},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from UpdateParticipantScore service"}
}

// HandleScoreBulkUpdateRequest handles bulk score overrides for a round.
func (h *RoundHandlers) HandleScoreBulkUpdateRequest(
	ctx context.Context,
	payload *roundevents.ScoreBulkUpdateRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, sharedtypes.ValidationError{Message: "bulk score update payload is nil"}
	}

	result, err := h.roundService.UpdateParticipantScoresBulk(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "bulk participant score update failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScoreUpdateErrorV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScoresBulkUpdatedV1, Payload: result.Success},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from UpdateParticipantScoresBulk service"}
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

	result, err := h.roundService.CheckAllScoresSubmitted(ctx, *payload)
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
