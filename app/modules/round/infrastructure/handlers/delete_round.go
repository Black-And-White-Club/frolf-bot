package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
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

	req := &roundtypes.DeleteRoundInput{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		UserID:  payload.RequestingUserUserID,
	}

	result, err := h.service.ValidateRoundDeletion(ctx, req)
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

	req := &roundtypes.DeleteRoundInput{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		// UserID not strictly needed for deletion op (already authorized), but available if needed?
		// AuthorizedPayload doesn't seem to have UserID.
		// DeleteRoundRequest struct has UserID.
		// Service.DeleteRound doesn't seem to check UserID (checked in Validate).
	}

	result, err := h.service.DeleteRound(ctx, req)
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

	// Add guild-scoped version for PWA permission scoping
	results = addGuildScopedResult(results, roundevents.RoundDeletedV1, payload.GuildID)

	return results, nil
}
