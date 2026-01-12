package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleDiscordMessageIDUpdated handles the event published after a round has been
// successfully updated with the Discord message ID and is ready for scheduling.
// It calls the service method to schedule the reminder and start events.
func (h *RoundHandlers) HandleDiscordMessageIDUpdated(
	ctx context.Context,
	payload *roundevents.RoundScheduledPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.roundService.ScheduleRoundEvents(ctx, payload.GuildID, *payload, payload.EventMessageID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round events scheduling failed in service",
			attr.RoundID("round_id", payload.RoundID),
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundScheduleFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		// Since this handler only schedules events and doesn't trigger downstream events,
		// we return an empty result slice to indicate successful processing.
		return []handlerwrapper.Result{}, nil
	}

	h.logger.ErrorContext(ctx, "unexpected empty result from ScheduleRoundEvents service",
		attr.RoundID("round_id", payload.RoundID),
	)
	return nil, sharedtypes.ValidationError{Message: "service returned neither success nor failure"}
}
