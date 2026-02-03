package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
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

	req := &roundtypes.UpdateRoundRequest{
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		StartTime: payload.StartTime,
	}
	if payload.Title != nil {
		t := string(*payload.Title)
		req.Title = &t
	}
	if payload.Description != nil {
		d := string(*payload.Description)
		req.Description = &d
	}
	if payload.Location != nil {
		l := string(*payload.Location)
		req.Location = &l
	}
	if payload.Timezone != nil {
		tz := string(*payload.Timezone)
		req.Timezone = &tz
	}

	result, err := h.service.ValidateRoundUpdateWithClock(ctx, req, roundtime.NewTimeParser(), clock)
	if err != nil {
		h.logger.ErrorContext(ctx, "ValidateRoundUpdateWithClock returned error",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
		return nil, err
	}

	// Map result to ensure correct event payload structure
	mappedResult := result.Map(
		func(r *roundtypes.UpdateRoundResult) any {
			var title *roundtypes.Title
			if req.Title != nil {
				t := roundtypes.Title(*req.Title)
				title = &t
			}
			var desc *roundtypes.Description
			if req.Description != nil {
				d := roundtypes.Description(*req.Description)
				desc = &d
			}
			var loc *roundtypes.Location
			if req.Location != nil {
				l := roundtypes.Location(*req.Location)
				loc = &l
			}

			// Use Round's StartTime if request had a start time update
			var startTime *sharedtypes.StartTime
			if req.StartTime != nil && r.Round != nil {
				startTime = r.Round.StartTime
			}

			return &roundevents.RoundUpdateValidatedPayloadV1{
				GuildID: req.GuildID,
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					GuildID:     req.GuildID,
					RoundID:     req.RoundID,
					UserID:      req.UserID,
					Title:       title,
					Description: desc,
					Location:    loc,
					StartTime:   startTime,
				},
			}
		},
		func(err error) any {
			// Construct payload for error case using outer 'req'
			var title *roundtypes.Title
			if req.Title != nil {
				t := roundtypes.Title(*req.Title)
				title = &t
			}
			var desc *roundtypes.Description
			if req.Description != nil {
				d := roundtypes.Description(*req.Description)
				desc = &d
			}
			var loc *roundtypes.Location
			if req.Location != nil {
				l := roundtypes.Location(*req.Location)
				loc = &l
			}

			requestPayload := roundevents.RoundUpdateRequestPayloadV1{
				GuildID:     req.GuildID,
				RoundID:     req.RoundID,
				UserID:      req.UserID,
				Title:       title,
				Description: desc,
				Location:    loc,
				// StartTime usage removed due to type mismatch
			}

			return &roundevents.RoundUpdateErrorPayloadV1{
				GuildID:            payload.GuildID,
				RoundUpdateRequest: &requestPayload,
				Error:              err.Error(),
			}
		},
	)

	if mappedResult.Failure != nil {
		h.logger.WarnContext(ctx, "round update validation failed",
			attr.RoundID("round_id", payload.RoundID),
			attr.Any("failure", mappedResult.Failure),
		)
		return mapOperationResult(mappedResult,
			roundevents.RoundUpdateValidatedV1,
			roundevents.RoundUpdateErrorV1,
		), nil
	}

	if mappedResult.Success != nil {
		h.logger.InfoContext(ctx, "round update validation successful, publishing validated event",
			attr.RoundID("round_id", payload.RoundID),
		)
		return mapOperationResult(mappedResult,
			roundevents.RoundUpdateValidatedV1,
			roundevents.RoundUpdateErrorV1,
		), nil
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

	req := &roundtypes.UpdateRoundRequest{
		GuildID:         payload.RoundUpdateRequestPayload.GuildID,
		RoundID:         payload.RoundUpdateRequestPayload.RoundID,
		UserID:          payload.RoundUpdateRequestPayload.UserID,
		ParsedStartTime: payload.RoundUpdateRequestPayload.StartTime,
	}
	if payload.RoundUpdateRequestPayload.Title != nil {
		t := string(*payload.RoundUpdateRequestPayload.Title)
		req.Title = &t
	}
	if payload.RoundUpdateRequestPayload.Description != nil {
		d := string(*payload.RoundUpdateRequestPayload.Description)
		req.Description = &d
	}
	if payload.RoundUpdateRequestPayload.Location != nil {
		l := string(*payload.RoundUpdateRequestPayload.Location)
		req.Location = &l
	}
	// Timezone is not available in validated payload but not needed since we have ParsedStartTime
	if payload.RoundUpdateRequestPayload.EventType != nil {
		req.EventType = payload.RoundUpdateRequestPayload.EventType
	}

	result, err := h.service.UpdateRoundEntity(ctx, req)
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
				Topic: roundevents.RoundUpdateErrorV1,
				Payload: &roundevents.RoundUpdateErrorPayloadV1{
					GuildID:            payload.RoundUpdateRequestPayload.GuildID,
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              (*result.Failure).Error(),
				},
			},
		}, nil
	}

	if result.Success != nil {
		// Use the result directly as it matches or map it if needed
		// The service returns UpdateRoundResult which has Round.
		// We need RoundEntityUpdatedPayloadV1.
		// Assuming we can construct it or the result is compatible (it's not).
		// We need to construct RoundEntityUpdatedPayloadV1.
		updatedPayload := &roundevents.RoundEntityUpdatedPayloadV1{
			GuildID: (*result.Success).Round.GuildID,
			Round:   *(*result.Success).Round,
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

		// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
		guildID := updatedPayload.GuildID
		if guildID == "" && updatedPayload.Round.GuildID != "" {
			guildID = updatedPayload.Round.GuildID
		}
		results = h.addParallelIdentityResults(ctx, results, roundevents.RoundUpdatedV1, guildID)

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

	titleStr := string(payload.Round.Title)
	locationStr := string(payload.Round.Location)

	req := &roundtypes.UpdateScheduledRoundEventsRequest{
		GuildID:   guildID,
		RoundID:   payload.Round.ID,
		Title:     &titleStr,
		StartTime: payload.Round.StartTime,
		Location:  &locationStr,
	}

	result, err := h.service.UpdateScheduledRoundEvents(ctx, req)
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
