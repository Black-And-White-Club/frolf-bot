package roundhandlers

import (
	"context"
	"errors"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
)

// HandleCreateRoundRequest handles the initial request to create a round.
func (h *RoundHandlers) HandleCreateRoundRequest(
	ctx context.Context,
	payload *roundevents.CreateRoundRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	clock := h.extractAnchorClock(ctx)

	result, err := h.service.ValidateAndProcessRoundWithClock(ctx, *payload, roundtime.NewTimeParser(), clock)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundEntityCreatedV1,
		roundevents.RoundValidationFailedV1,
	), nil
}

// HandleRoundEntityCreated handles persisting the round entity to the database.
func (h *RoundHandlers) HandleRoundEntityCreated(
	ctx context.Context,
	payload *roundevents.RoundEntityCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.StoreRound(ctx, payload.GuildID, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundCreatedV1,
		roundevents.RoundCreationFailedV1,
	), nil
}

// HandleRoundEventMessageIDUpdate updates the round with the Discord message ID.
// compatibility with the Discord module on Main.
func (h *RoundHandlers) HandleRoundEventMessageIDUpdate(
	ctx context.Context,
	payload *roundevents.RoundMessageIDUpdatePayloadV1,
) ([]handlerwrapper.Result, error) {
	// 1. Extract metadata injected into context by the wrapper
	discordMessageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || discordMessageID == "" {
		return nil, errors.New("discord_message_id missing from context")
	}

	// 2. Call service to persist the ID
	updatedRound, err := h.service.UpdateRoundMessageID(ctx, payload.GuildID, payload.RoundID, discordMessageID)
	if err != nil {
		return nil, err
	}

	if updatedRound == nil {
		return nil, errors.New("updated round object is nil")
	}

	// 3. Construct outgoing payload
	scheduledPayload := roundevents.RoundScheduledPayloadV1{
		GuildID: payload.GuildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     updatedRound.ID,
			Title:       updatedRound.Title,
			Description: updatedRound.Description,
			Location:    updatedRound.Location,
			StartTime:   updatedRound.StartTime,
			UserID:      updatedRound.CreatedBy,
		},
		EventMessageID: discordMessageID,
	}

	// 4. Explicitly promote the metadata to the outgoing message headers.
	// Without this, the Discord Module will not know which message to track.
	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundEventMessageIDUpdatedV1,
			Payload: scheduledPayload,
			Metadata: map[string]string{
				"discord_message_id": discordMessageID,
			},
		},
	}, nil
}
