package roundservice

import (
	"context"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
// It handles cases where the round start time might be too close for certain reminders.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (ScheduleRoundEventsResult, error) {
	return withTelemetry[*roundtypes.ScheduleRoundEventsResult, error](s, ctx, "ScheduleRoundEvents", req.RoundID, func(ctx context.Context) (ScheduleRoundEventsResult, error) {
		s.logger.InfoContext(ctx, "Processing round scheduling",
			attr.RoundID("round_id", req.RoundID),
			attr.Time("start_time", req.StartTime.AsTime()),
		)

		// Cancel any existing scheduled jobs for this round
		s.logger.InfoContext(ctx, "Cancelling existing scheduled jobs",
			attr.RoundID("round_id", req.RoundID),
		)

		if err := s.queueService.CancelRoundJobs(ctx, req.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to cancel existing jobs",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
			return results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{}, err
		}

		// Calculate times
		now := time.Now().UTC()
		startTimeUTC := req.StartTime.AsTime().UTC()
		reminderTimeUTC := startTimeUTC.Add(-1 * time.Hour)

		// Schedule reminder if there's enough time (at least 5 minutes in the future)
		if reminderTimeUTC.After(now.Add(5 * time.Second)) {
			s.logger.InfoContext(ctx, "Scheduling 1-hour reminder",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)

			reminderPayload := roundevents.DiscordReminderPayloadV1{
				GuildID:        req.GuildID, // Multi-tenant scope required downstream
				RoundID:        req.RoundID,
				ReminderType:   "1h",
				RoundTitle:     roundtypes.Title(req.Title),
				Location:       roundtypes.Location(req.Location),
				StartTime:      &req.StartTime,
				EventMessageID: req.EventMessageID,
				// DiscordGuildID duplicates GuildID as raw string for Discord service convenience
				DiscordGuildID: string(req.GuildID),
			}

			// Channel enrichment precedence:
			// 1. Config fragment event channel
			// 2. ChannelID present on scheduling payload (if RoundScheduledPayload now carries it)
			// 3. Guild config provider
			if req.Config != nil && req.Config.EventChannelID != "" {
				reminderPayload.DiscordChannelID = req.Config.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from payload.Config)", attr.String("channel_id", req.Config.EventChannelID))
			} else if req.ChannelID != "" {
				reminderPayload.DiscordChannelID = req.ChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from payload.ChannelID)", attr.String("channel_id", req.ChannelID))
			} else if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil && cfg.EventChannelID != "" {
				reminderPayload.DiscordChannelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from guild config provider)", attr.String("channel_id", cfg.EventChannelID))
			} else {
				s.logger.WarnContext(ctx, "No event channel ID available to embed in reminder payload", attr.String("guild_id", string(req.GuildID)))
			}

			if err := s.queueService.ScheduleRoundReminder(ctx, req.GuildID, req.RoundID, reminderTimeUTC, reminderPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule reminder job",
					attr.RoundID("round_id", req.RoundID),
					attr.Error(err),
				)
				return results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{}, err
			}
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder - not enough time",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		}

		// Schedule round start if in the future (at least 5 seconds buffer)
		if startTimeUTC.After(now.Add(5 * time.Second)) {
			s.logger.InfoContext(ctx, "Scheduling round start",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
			)

			startPayload := roundevents.RoundStartedPayloadV1{
				GuildID:   req.GuildID, // Ensure downstream start processing has tenant scope
				RoundID:   req.RoundID,
				Title:     roundtypes.Title(req.Title),
				Location:  roundtypes.Location(req.Location),
				StartTime: &req.StartTime,
			}

			if err := s.queueService.ScheduleRoundStart(ctx, req.GuildID, req.RoundID, startTimeUTC, startPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule round start job",
					attr.RoundID("round_id", req.RoundID),
					attr.Error(err),
				)
				return results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{}, err
			}
		} else {
			s.logger.InfoContext(ctx, "Round start time is too close or in the past, not scheduling",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
			)
		}

		// Return success with the original payload
		return results.SuccessResult[*roundtypes.ScheduleRoundEventsResult, error](&roundtypes.ScheduleRoundEventsResult{
			RoundID:        req.RoundID,
			GuildID:        req.GuildID,
			Title:          req.Title,
			Description:    req.Description,
			Location:       req.Location,
			StartTime:      req.StartTime,
			EventMessageID: req.EventMessageID,
			ChannelID:      req.ChannelID,
		}), nil
	})
}
