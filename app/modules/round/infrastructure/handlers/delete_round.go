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

	mapped := result.Map(
		func(_ *roundtypes.Round) any {
			return &roundevents.RoundDeleteValidatedPayloadV1{
				GuildID: payload.GuildID,
				RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              payload.GuildID,
					RoundID:              payload.RoundID,
					RequestingUserUserID: payload.RequestingUserUserID,
				},
			}
		},
		func(err error) any {
			return &roundevents.RoundDeleteErrorPayloadV1{
				GuildID: payload.GuildID,
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              payload.GuildID,
					RoundID:              payload.RoundID,
					RequestingUserUserID: payload.RequestingUserUserID,
				},
				Error: err.Error(),
			}
		},
	)

	return mapOperationResult(mapped,
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
	// 2. Fetch DiscordEventID before deletion (best-effort, non-blocking)
	var discordEventID string
	roundResult, err := h.service.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err == nil && roundResult.Success != nil {
		discordEventID = (*roundResult.Success).DiscordEventID
	}

	req := &roundtypes.DeleteRoundInput{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
	}

	result, err := h.service.DeleteRound(ctx, req)
	if err != nil {
		return nil, err
	}

	// Map to ensure correct event payload structure
	mappedResult := result.Map(
		func(_ bool) any {
			// Extract Discord IDs from context if available (set by wrapper from message metadata)
			discordMessageID, _ := ctx.Value("discord_message_id").(string)
			channelID, _ := ctx.Value("channel_id").(string)

			return &roundevents.RoundDeletedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ChannelID:      channelID,
				EventMessageID: discordMessageID,
				DiscordEventID: discordEventID,
			}
		},
		func(err error) any {
			return &roundevents.RoundDeleteErrorPayloadV1{
				GuildID: payload.GuildID,
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
				},
				Error: err.Error(),
			}
		},
	)

	results := mapOperationResult(mappedResult,
		roundevents.RoundDeletedV1,
		roundevents.RoundDeleteErrorV1,
	)

	// Propagate discord_message_id to metadata if present in context
	if discordMessageID, ok := ctx.Value("discord_message_id").(string); ok && discordMessageID != "" {
		for i := range results {
			if results[i].Metadata == nil {
				results[i].Metadata = make(map[string]string)
			}
			results[i].Metadata["discord_message_id"] = discordMessageID
		}
	}

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundDeletedV1, payload.GuildID)

	return results, nil
}
