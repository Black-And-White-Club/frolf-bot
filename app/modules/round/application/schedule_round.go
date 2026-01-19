package roundservice

import (
	"context"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
// It handles cases where the round start time might be too close for certain reminders.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundScheduledPayloadV1, discordMessageID string) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ScheduleRoundEvents", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round scheduling",
			attr.RoundID("round_id", payload.RoundID),
			attr.Time("start_time", payload.StartTime.AsTime()),
		)

		// Cancel any existing scheduled jobs for this round
		s.logger.InfoContext(ctx, "Cancelling existing scheduled jobs",
			attr.RoundID("round_id", payload.RoundID),
		)

		if err := s.queueService.CancelRoundJobs(ctx, payload.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to cancel existing jobs",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Calculate times
		now := time.Now().UTC()
		startTimeUTC := payload.StartTime.AsTime().UTC()
		reminderTimeUTC := startTimeUTC.Add(-1 * time.Hour)

		// Schedule reminder if there's enough time (at least 5 minutes in the future)
		if reminderTimeUTC.After(now.Add(5 * time.Second)) {
			s.logger.InfoContext(ctx, "Scheduling 1-hour reminder",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)

			reminderPayload := roundevents.DiscordReminderPayloadV1{
				GuildID:        guildID, // Multi-tenant scope required downstream
				RoundID:        payload.RoundID,
				ReminderType:   "1h",
				RoundTitle:     payload.Title,
				Location:       payload.Location,
				StartTime:      payload.StartTime,
				EventMessageID: discordMessageID,
				// DiscordGuildID duplicates GuildID as raw string for Discord service convenience
				DiscordGuildID: string(guildID),
			}

			// Channel enrichment precedence:
			// 1. Config fragment event channel
			// 2. ChannelID present on scheduling payload (if RoundScheduledPayload now carries it)
			// 3. Guild config provider
			if payload.Config != nil && payload.Config.EventChannelID != "" {
				reminderPayload.DiscordChannelID = payload.Config.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from payload.Config)", attr.String("channel_id", payload.Config.EventChannelID))
			} else if payload.ChannelID != "" {
				reminderPayload.DiscordChannelID = payload.ChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from payload.ChannelID)", attr.String("channel_id", payload.ChannelID))
			} else if cfg := s.getGuildConfigForEnrichment(ctx, guildID); cfg != nil && cfg.EventChannelID != "" {
				reminderPayload.DiscordChannelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into reminder payload (from guild config provider)", attr.String("channel_id", cfg.EventChannelID))
			} else {
				s.logger.WarnContext(ctx, "No event channel ID available to embed in reminder payload", attr.String("guild_id", string(guildID)))
			}

			if err := s.queueService.ScheduleRoundReminder(ctx, guildID, payload.RoundID, reminderTimeUTC, reminderPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule reminder job",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return results.OperationResult{
					Failure: &roundevents.RoundErrorPayloadV1{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, nil
			}
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder - not enough time",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		}

		// Schedule round start if in the future (at least 5 seconds buffer)
		if startTimeUTC.After(now.Add(5 * time.Second)) {
			s.logger.InfoContext(ctx, "Scheduling round start",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
			)

			startPayload := roundevents.RoundStartedPayloadV1{
				GuildID:   guildID, // Ensure downstream start processing has tenant scope
				RoundID:   payload.RoundID,
				Title:     payload.Title,
				Location:  payload.Location,
				StartTime: payload.StartTime,
			}

			if err := s.queueService.ScheduleRoundStart(ctx, guildID, payload.RoundID, startTimeUTC, startPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule round start job",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return results.OperationResult{
					Failure: &roundevents.RoundErrorPayloadV1{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, nil
			}
		} else {
			s.logger.InfoContext(ctx, "Round start time is too close or in the past, not scheduling",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
			)
		}

		// Return success with the original payload
		return results.OperationResult{
			Success: &roundevents.RoundScheduledPayloadV1{
				GuildID: guildID, // ensure guild scope propagates for downstream handlers
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     payload.RoundID,
					Title:       payload.Title,
					Description: payload.Description,
					Location:    payload.Location,
					StartTime:   payload.StartTime,
				},
				EventMessageID: discordMessageID,
				ChannelID:      payload.ChannelID, // propagate for downstream enrichment
			},
		}, nil
	})
}
