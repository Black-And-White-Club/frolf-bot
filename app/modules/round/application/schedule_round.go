package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
// It handles cases where the round start time might be too close for certain reminders.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, payload roundevents.RoundScheduledPayload, discordMessageID string) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ScheduleRoundEvents", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Scheduling round events",
			attr.RoundID("round_id", payload.RoundID),
		)

		// Ensure the consumer is created for this round ID
		if err := s.EventBus.ProcessDelayedMessages(ctx, payload.RoundID, *payload.StartTime); err != nil {
			s.logger.ErrorContext(ctx, "Failed to create consumer for round",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to create consumer for round %s: %w", payload.RoundID, err)
		}

		// Get current time to evaluate which events to schedule
		now := time.Now().UTC()

		// Calculate reminder time (1 hour before the round starts)
		reminderTime := payload.StartTime.Add(-1 * time.Hour)

		// Only schedule reminder if there's enough time (reminder time is in the future)
		if reminderTime.AsTime().After(now) {
			s.logger.InfoContext(ctx, "Scheduling 1-hour reminder",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("reminder_time", reminderTime.AsTime()),
			)

			// Prepare reminder payload
			reminderPayload := roundevents.DiscordReminderPayload{
				RoundID:        payload.RoundID,
				ReminderType:   "1h",
				RoundTitle:     payload.Title,
				EventMessageID: discordMessageID,
			}
			reminderBytes, err := json.Marshal(reminderPayload)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to encode reminder payload",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: roundevents.RoundErrorPayload{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, fmt.Errorf("failed to encode reminder payload: %w", err)
			}

			// Schedule reminder with empty additional metadata map
			additionalMetadata := make(map[string]string)
			if err := s.EventBus.ScheduleDelayedMessage(ctx, roundevents.RoundReminder, payload.RoundID, reminderTime, reminderBytes, additionalMetadata); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule reminder",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: roundevents.RoundErrorPayload{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, fmt.Errorf("failed to schedule reminder: %w", err)
			}
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder - not enough time before round start",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", payload.StartTime.AsTime()),
				attr.Time("current_time", now),
			)
		}

		// Always schedule the round start if it's in the future
		if payload.StartTime.AsTime().After(now) {
			s.logger.InfoContext(ctx, "Scheduling round start event",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", payload.StartTime.AsTime()),
			)

			// Prepare round start payload
			startPayload := roundevents.DiscordRoundStartPayload{
				RoundID:        payload.RoundID,
				Title:          payload.Title,
				Location:       payload.Location,
				StartTime:      payload.StartTime,
				Participants:   []roundevents.RoundParticipant{},
				EventMessageID: discordMessageID,
			}
			startBytes, err := json.Marshal(startPayload)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to encode round start payload",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: roundevents.RoundErrorPayload{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, fmt.Errorf("failed to encode round start payload: %w", err)
			}

			// Schedule start event with empty additional metadata map
			additionalMetadata := make(map[string]string)
			if err := s.EventBus.ScheduleDelayedMessage(ctx, roundevents.RoundStarted, payload.RoundID, *payload.StartTime, startBytes, additionalMetadata); err != nil {
				s.logger.ErrorContext(ctx, "Failed to schedule round start",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: roundevents.RoundErrorPayload{
						RoundID: payload.RoundID,
						Error:   err.Error(),
					},
				}, fmt.Errorf("failed to schedule round start: %w", err)
			}
		} else {
			s.logger.WarnContext(ctx, "Round start time is in the past, not scheduling start event",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", payload.StartTime.AsTime()),
				attr.Time("current_time", now),
			)
		}

		s.logger.InfoContext(ctx, "Round events scheduled",
			attr.RoundID("round_id", payload.RoundID),
		)

		scheduledPayload := roundevents.RoundScheduledPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     payload.RoundID,
				Title:       payload.Title,
				Description: payload.Description,
				Location:    payload.Location,
				StartTime:   payload.StartTime,
			},
			EventMessageID: discordMessageID,
		}

		return RoundOperationResult{
			Success: scheduledPayload,
		}, nil
	})
}
