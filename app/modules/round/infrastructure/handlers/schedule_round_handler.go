package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleDiscordMessageIDUpdated handles the event published after a round has been
// successfully updated with the Discord message ID and is ready for scheduling.
// It calls the service method to schedule the reminder and start events.
func (h *RoundHandlers) HandleDiscordMessageIDUpdated(
	ctx context.Context,
	payload *roundevents.RoundScheduledPayloadV1,
) ([]handlerwrapper.Result, error) {
	var config *guildtypes.GuildConfig
	if payload.Config != nil {
		config = &guildtypes.GuildConfig{
			EventChannelID: payload.Config.EventChannelID,
		}
	}

	result, err := h.service.ScheduleRoundEvents(ctx, &roundtypes.ScheduleRoundEventsRequest{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		Title:          payload.Title.String(),
		Description:    payload.Description.String(),
		Location:       payload.Location.String(),
		StartTime:      *payload.StartTime,
		UserID:         payload.UserID,
		EventMessageID: payload.EventMessageID,
		ChannelID:      payload.ChannelID,
		Config:         config,
	})
	if err != nil {
		return nil, err
	}

	// Since this handler only schedules events and doesn't trigger downstream events,
	// we consume the result and return empty results to indicate successful processing.
	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round events scheduling failed in service",
			attr.RoundID("round_id", payload.RoundID),
			attr.Any("failure", *result.Failure),
		)
		return []handlerwrapper.Result{}, nil
	}

	return []handlerwrapper.Result{}, nil
}
