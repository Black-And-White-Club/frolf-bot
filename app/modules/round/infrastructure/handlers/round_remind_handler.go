package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundReminder processes incoming reminder requests from the scheduler.
// It transforms the request into a formatted Discord reminder payload via the round service.
func (h *RoundHandlers) HandleRoundReminder(
	ctx context.Context,
	payload *roundevents.DiscordReminderPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Diagnostic logging to ensure the reminder flow has the necessary context (Guild/Round/Type).
	h.logger.DebugContext(ctx, "processing round reminder request",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reminder_type", payload.ReminderType),
	)

	// Delegate to the service to fetch participant lists and format the final notification data.
	result, err := h.roundService.ProcessRoundReminder(ctx, *payload)
	if err != nil {
		return nil, err
	}

	// Handle functional failures (e.g., round no longer exists or is cancelled).
	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round reminder processing failed in service",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundReminderFailedV1, Payload: result.Failure},
		}, nil
	}

	// Handle successful preparation of the reminder.
	if result.Success != nil {
		discordPayload, ok := result.Success.(*roundevents.DiscordReminderPayloadV1)
		if !ok {
			return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from ProcessRoundReminder"}
		}

		// Only publish to the Discord module if there is actually someone to notify.
		// This prevents "empty" pings or spamming channels if no one joined the round.
		if len(discordPayload.UserIDs) > 0 {
			h.logger.InfoContext(ctx, "round reminder ready for delivery",
				attr.RoundID("round_id", payload.RoundID),
				attr.Int("recipient_count", len(discordPayload.UserIDs)),
			)
			return []handlerwrapper.Result{
				{Topic: roundevents.RoundReminderSentV1, Payload: discordPayload},
			}, nil
		}

		// Success, but no participants joined, so no message needs to be sent to Discord.
		h.logger.InfoContext(ctx, "skipping reminder delivery: no participants to notify",
			attr.RoundID("round_id", payload.RoundID),
		)
		return []handlerwrapper.Result{}, nil
	}

	h.logger.ErrorContext(ctx, "service returned empty result for round reminder")
	return nil, sharedtypes.ValidationError{Message: "service returned neither success nor failure"}
}
