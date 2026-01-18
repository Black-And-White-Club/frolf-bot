package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
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

	result, err := h.service.ValidateRoundDeleteRequest(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundDeleteValidatedV1,
		roundevents.RoundDeleteErrorV1,
	), nil
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
	discordMessageID, _ := ctx.Value("discord_message_id").(string)

	result, err := h.service.DeleteRound(ctx, *payload)
	if err != nil {
		return nil, err
	}

	results := mapOperationResult(result,
		roundevents.RoundDeletedV1,
		roundevents.RoundDeleteErrorV1,
	)

	// 2. If we have a message ID, promote it to metadata so the Discord
	// handler knows which embed to delete.
	if discordMessageID != "" && len(results) > 0 {
		results[0].Metadata = map[string]string{
			"discord_message_id": discordMessageID,
		}
	}

	return results, nil
}
