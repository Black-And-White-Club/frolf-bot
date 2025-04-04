package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, payload roundevents.RoundStoredPayload, startTime sharedtypes.StartTime) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ScheduleRoundEvents", func() (RoundOperationResult, error) {
		s.logger.Info("Scheduling round events",
			attr.RoundID("round_id", payload.Round.ID),
		)

		// Ensure the consumer is created for this round ID
		if err := s.EventBus.ProcessDelayedMessages(ctx, payload.Round.ID, startTime); err != nil {
			s.logger.Error("Failed to create consumer for round",
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.Round.ID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to create consumer for round %s: %w", payload.Round.ID, err)
		}

		// Schedule reminder
		reminderPayload := roundevents.DiscordReminderPayload{
			RoundID:        payload.Round.ID,
			ReminderType:   "1h",
			RoundTitle:     payload.Round.Title,
			EventMessageID: payload.Round.EventMessageID,
		}
		reminderBytes, err := json.Marshal(reminderPayload)
		if err != nil {
			s.logger.Error("Failed to encode reminder payload",
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.Round.ID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to encode reminder payload: %w", err)
		}
		reminderTime := startTime.Add(-1 * time.Hour) // Reminder time is 1 hour before the round starts
		if err := s.EventBus.ScheduleDelayedMessage(ctx, roundevents.RoundReminder, payload.Round.ID, reminderTime, reminderBytes); err != nil {
			s.logger.Error("Failed to schedule reminder",
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.Round.ID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to schedule reminder: %w", err)
		}

		// Schedule round start
		startPayload := roundevents.DiscordRoundStartPayload{
			RoundID:        payload.Round.ID,
			Title:          payload.Round.Title,
			Location:       payload.Round.Location,
			StartTime:      (*sharedtypes.StartTime)(&startTime),
			Participants:   []roundevents.RoundParticipant{},
			EventMessageID: payload.Round.EventMessageID,
		}
		startBytes, err := json.Marshal(startPayload)
		if err != nil {
			s.logger.Error("Failed to encode round start payload",
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.Round.ID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to encode round start payload: %w", err)
		}
		if err := s.EventBus.ScheduleDelayedMessage(ctx, roundevents.RoundStarted, payload.Round.ID, startTime, startBytes); err != nil {
			s.logger.Error("Failed to schedule round start",
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.Round.ID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to schedule round start: %w", err)
		}

		s.logger.Info("Round events scheduled",
			attr.RoundID("round_id", payload.Round.ID),
		)

		scheduledPayload := roundevents.RoundScheduledPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     payload.Round.ID,
				Title:       payload.Round.Title,
				Description: payload.Round.Description,
				Location:    payload.Round.Location,
				StartTime:   (*sharedtypes.StartTime)(&startTime),
				UserID:      payload.Round.CreatedBy,
			},
			EventMessageID: &payload.Round.EventMessageID,
		}

		return RoundOperationResult{
			Success: scheduledPayload,
		}, nil
	})
}
