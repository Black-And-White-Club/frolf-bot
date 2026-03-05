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

		hasNativeDiscordEvent := false
		nativeEventLookupFailed := false
		if s.repo != nil {
			storedRound, lookupErr := s.repo.GetRound(ctx, s.db, req.GuildID, req.RoundID)
			if lookupErr != nil {
				nativeEventLookupFailed = true
				s.logger.WarnContext(ctx, "Failed to inspect round native event linkage; defaulting to queue-based start fallback",
					attr.RoundID("round_id", req.RoundID),
					attr.Error(lookupErr),
				)
			} else if storedRound != nil && storedRound.DiscordEventID != "" {
				hasNativeDiscordEvent = true
			}
		} else {
			s.logger.WarnContext(ctx, "Round repository unavailable during scheduling; using queue-based start fallback",
				attr.RoundID("round_id", req.RoundID),
			)
		}

		s.logger.DebugContext(ctx, "Round start scheduling decision",
			attr.RoundID("round_id", req.RoundID),
			attr.Bool("has_native_discord_event", hasNativeDiscordEvent),
			attr.Bool("event_message_id_present", req.EventMessageID != ""),
			attr.Bool("native_event_lookup_failed", nativeEventLookupFailed),
		)

		// Schedule reminder only when we have a Discord message context and enough lead time.
		if req.EventMessageID != "" && reminderTimeUTC.After(now.Add(5*time.Second)) {
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
		} else if req.EventMessageID == "" {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder - no event message id (PWA-only or pre-Discord round)",
				attr.RoundID("round_id", req.RoundID),
			)
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder - not enough time",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		}

		// Schedule round start if in the future (at least 5 seconds buffer) and no
		// Discord native scheduled event is linked yet. Linked Discord events remain
		// authoritative for lifecycle start transitions.
		if startTimeUTC.After(now.Add(5*time.Second)) && !hasNativeDiscordEvent {
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

			if req.ChannelID != "" {
				startPayload.ChannelID = req.ChannelID
			} else if req.Config != nil && req.Config.EventChannelID != "" {
				startPayload.ChannelID = req.Config.EventChannelID
			} else if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil && cfg.EventChannelID != "" {
				startPayload.ChannelID = cfg.EventChannelID
			}

			if err := s.queueService.ScheduleRoundStart(ctx, req.GuildID, req.RoundID, startTimeUTC, startPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule round start job",
					attr.RoundID("round_id", req.RoundID),
					attr.Error(err),
				)
				return results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{}, err
			}
		} else if hasNativeDiscordEvent {
			s.logger.InfoContext(ctx, "Skipping queue-based round start scheduling - Discord native event flow will trigger start",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
			)
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
