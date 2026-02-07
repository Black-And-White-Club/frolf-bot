package roundhandlers

import (
	"context"
	"errors"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// HandleNativeEventCreated handles storing the Discord Native Event ID
// when a Guild Scheduled Event is successfully created for a round.
// This is a terminal sink handler that updates the round's discord_event_id field.
func (h *RoundHandlers) HandleNativeEventCreated(
	ctx context.Context,
	payload *roundevents.NativeEventCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Call service to persist the Discord Event ID
	updatedRound, err := h.service.UpdateDiscordEventID(ctx, payload.GuildID, payload.RoundID, payload.DiscordEventID)
	if err != nil {
		return nil, err
	}

	if updatedRound == nil {
		return nil, errors.New("updated round object is nil")
	}

	// This is a terminal sink handler - no outgoing events
	return []handlerwrapper.Result{}, nil
}

// HandleNativeEventLookupRequest handles resolving a DiscordEventID to a RoundID.
// This is used as a fallback when the discord-service's in-memory map is empty
// (e.g., after bot restart) and it needs to resolve RSVP events.
func (h *RoundHandlers) HandleNativeEventLookupRequest(
	ctx context.Context,
	payload *roundevents.NativeEventLookupRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Call service to look up the round by Discord Event ID
	round, err := h.service.GetRoundByDiscordEventID(ctx, payload.GuildID, payload.DiscordEventID)

	// If a technical error occurred (DB down, etc), return error to trigger retry.
	// We only return success (Found=false) if we specifically know it wasn't found.
	if err != nil && !errors.Is(err, roundservice.ErrRoundNotFound) {
		return nil, err
	}

	// Build result payload
	resultPayload := &roundevents.NativeEventLookupResultPayloadV1{
		GuildID:        payload.GuildID,
		DiscordEventID: payload.DiscordEventID,
		Found:          false,
	}

	// If round found, populate the result
	if err == nil && round != nil {
		resultPayload.RoundID = round.ID
		resultPayload.Found = true
	}

	// Return the lookup result (Found=true or Found=false)
	return []handlerwrapper.Result{
		{
			Topic:   roundevents.NativeEventLookupResultV1,
			Payload: resultPayload,
		},
	}, nil
}
