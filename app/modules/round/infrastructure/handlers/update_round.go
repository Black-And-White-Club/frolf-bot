package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
)

// HandleRoundUpdateRequest handles the initial validation of a round update request.
// It uses the Pure Transformer pattern: logic depends on context-injected clock,
// and metadata propagation is handled by the router's wrapper.
func (h *RoundHandlers) HandleRoundUpdateRequest(
	ctx context.Context,
	payload *roundevents.UpdateRoundRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "HandleRoundUpdateRequest called",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("description", func() string {
			if payload.Description != nil {
				return string(*payload.Description)
			}
			return "<nil>"
		}()),
	)

	clock := h.extractAnchorClock(ctx)

	result, err := h.service.ValidateAndProcessRoundUpdateWithClock(ctx, *payload, roundtime.NewTimeParser(), clock)
	if err != nil {
		h.logger.ErrorContext(ctx, "ValidateAndProcessRoundUpdateWithClock returned error",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round update validation failed",
			attr.RoundID("round_id", payload.RoundID),
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundUpdateErrorV1,
				Payload: result.Failure,
			},
		}, nil
	}

	if result.Success != nil {
		h.logger.InfoContext(ctx, "round update validation successful, publishing validated event",
			attr.RoundID("round_id", payload.RoundID),
		)
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundUpdateValidatedV1,
				Payload: result.Success,
			},
		}, nil
	}

	h.logger.ErrorContext(ctx, "ValidateAndProcessRoundUpdateWithClock returned empty result",
		attr.RoundID("round_id", payload.RoundID),
	)
	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from ValidateAndProcessRoundUpdate service"}
}

// HandleRoundUpdateValidated applies the validated update to the round entity.
func (h *RoundHandlers) HandleRoundUpdateValidated(
	ctx context.Context,
	payload *roundevents.RoundUpdateValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "HandleRoundUpdateValidated called",
		attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		attr.String("description", func() string {
			if payload.RoundUpdateRequestPayload.Description != nil {
				return string(*payload.RoundUpdateRequestPayload.Description)
			}
			return "<nil>"
		}()),
	)

	result, err := h.service.UpdateRoundEntity(ctx, *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdateRoundEntity returned error",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round entity update failed",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundUpdateErrorV1,
				Payload: result.Failure,
			},
		}, nil
	}

	if result.Success != nil {
		updatedPayload, ok := result.Success.(*roundevents.RoundEntityUpdatedPayloadV1)
		if !ok {
			return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from UpdateRoundEntity"}
		}

		h.logger.InfoContext(ctx, "round entity update successful, publishing results",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		)

		results := []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundUpdatedV1,
				Payload: updatedPayload,
			},
		}

		// Add guild-scoped version for PWA permission scoping
		guildID := updatedPayload.GuildID
		if guildID == "" && updatedPayload.Round.GuildID != "" {
			guildID = updatedPayload.Round.GuildID
		}
		results = addGuildScopedResult(results, roundevents.RoundUpdatedV1, guildID)

		// Check if we need to reschedule (only for time-sensitive fields)
		if h.shouldRescheduleEvents(payload.RoundUpdateRequestPayload) {
			h.logger.InfoContext(ctx, "scheduling round reschedule event",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			)
			results = append(results, handlerwrapper.Result{
				Topic:   roundevents.RoundScheduleUpdatedV1,
				Payload: updatedPayload,
			})
		}

		return results, nil
	}

	h.logger.ErrorContext(ctx, "UpdateRoundEntity returned empty result",
		attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
	)
	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from UpdateRoundEntity service"}
}

// HandleRoundScheduleUpdate manages updating downstream scheduled events (reminders, etc.)
func (h *RoundHandlers) HandleRoundScheduleUpdate(
	ctx context.Context,
	payload *roundevents.RoundEntityUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	guildID := payload.Round.GuildID
	if guildID == "" {
		guildID = payload.GuildID
	}

	schedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
		GuildID:   guildID,
		RoundID:   payload.Round.ID,
		Title:     payload.Round.Title,
		StartTime: payload.Round.StartTime,
		Location:  payload.Round.Location,
	}

	result, err := h.service.UpdateScheduledRoundEvents(ctx, schedulePayload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "scheduled round update failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundUpdateErrorV1,
				Payload: result.Failure,
			},
		}, nil
	}

	if result.Success != nil {
		// Successful rescheduling usually doesn't require a downstream event
		return []handlerwrapper.Result{}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from UpdateScheduledRoundEvents service"}
}

// shouldRescheduleEvents determines if a round update requires event rescheduling.
func (h *RoundHandlers) shouldRescheduleEvents(payload roundevents.RoundUpdateRequestPayloadV1) bool {
	return payload.StartTime != nil
}
