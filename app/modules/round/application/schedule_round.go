package roundservice

import (
	"context"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
// It handles cases where the round start time might be too close for certain reminders.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, payload roundevents.RoundScheduledPayload, discordMessageID string) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ScheduleRoundEvents", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round scheduling",
			attr.RoundID("round_id", payload.RoundID),
			attr.Time("start_time", payload.StartTime.AsTime()),
		)

		// Cancel any existing scheduled jobs for this round
		s.logger.InfoContext(ctx, "Cancelling existing scheduled jobs",
			attr.RoundID("round_id", payload.RoundID),
		)

		if err := s.QueueService.CancelRoundJobs(ctx, payload.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to cancel existing jobs",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
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

			reminderPayload := roundevents.DiscordReminderPayload{
				RoundID:        payload.RoundID,
				ReminderType:   "1h",
				RoundTitle:     payload.Title,
				Location:       payload.Location,
				StartTime:      payload.StartTime,
				EventMessageID: discordMessageID,
			}

			if err := s.QueueService.ScheduleRoundReminder(ctx, payload.RoundID, reminderTimeUTC, reminderPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule reminder job",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: &roundevents.RoundErrorPayload{
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

		// Schedule round start if in the future (at least 30 seconds buffer)
		if startTimeUTC.After(now.Add(30 * time.Second)) {
			s.logger.InfoContext(ctx, "Scheduling round start",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
			)

			startPayload := roundevents.RoundStartedPayload{
				RoundID:   payload.RoundID,
				Title:     payload.Title,
				Location:  payload.Location,
				StartTime: payload.StartTime,
			}

			if err := s.QueueService.ScheduleRoundStart(ctx, payload.RoundID, startTimeUTC, startPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule round start job",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: &roundevents.RoundErrorPayload{
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
		return RoundOperationResult{
			Success: &roundevents.RoundScheduledPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     payload.RoundID,
					Title:       payload.Title,
					Description: payload.Description,
					Location:    payload.Location,
					StartTime:   payload.StartTime,
				},
				EventMessageID: discordMessageID,
			},
		}, nil
	})
}
