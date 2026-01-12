package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleRoundDeleteRequest handles the initial request to delete a round.
func (h *RoundHandlers) HandleRoundDeleteRequest(
	ctx context.Context,
	payload *roundevents.RoundDeleteRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Pre-validation for safety
	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		return nil, sharedtypes.ValidationError{Message: "invalid round ID: cannot process delete request with nil UUID"}
	}

	result, err := h.roundService.ValidateRoundDeleteRequest(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round delete request validation failed", attr.Any("failure", result.Failure))
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundDeleteErrorV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundDeleteValidatedV1, Payload: result.Success},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "service returned unexpected nil result from ValidateRoundDeleteRequest"}
}

// HandleRoundDeleteValidated moves the process forward once validation is complete.
func (h *RoundHandlers) HandleRoundDeleteValidated(
	ctx context.Context,
	payload *roundevents.RoundDeleteValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Simple transformation to the authorized state.
	authorizedPayload := &roundevents.RoundDeleteAuthorizedPayloadV1{
		GuildID: payload.RoundDeleteRequestPayload.GuildID,
		RoundID: payload.RoundDeleteRequestPayload.RoundID,
	}

	return []handlerwrapper.Result{
		{Topic: roundevents.RoundDeleteAuthorizedV1, Payload: authorizedPayload},
	}, nil
}

// HandleRoundDeleteAuthorized executes the final deletion after authorization.
func (h *RoundHandlers) HandleRoundDeleteAuthorized(
	ctx context.Context,
	payload *roundevents.RoundDeleteAuthorizedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// 1. Extract the Discord Message ID from context to ensure it propagates
	discordMessageID, _ := ctx.Value("message_id").(string)

	result, err := h.roundService.DeleteRound(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round delete execution failed",
			attr.RoundID("round_id", payload.RoundID),
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundDeleteErrorV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		res := handlerwrapper.Result{
			Topic:   roundevents.RoundDeletedV1,
			Payload: result.Success,
		}

		// 2. If we have a message ID, promote it to metadata so the Discord
		// handler knows which embed to delete.
		if discordMessageID != "" {
			res.Metadata = map[string]string{
				"message_id": discordMessageID,
			}
		}

		return []handlerwrapper.Result{res}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "service returned unexpected nil result from DeleteRound"}
}
