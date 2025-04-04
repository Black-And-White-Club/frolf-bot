package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundReminder handles the reminder event when it's triggered from the delayed queue
func (s *RoundService) ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ProcessRoundReminder", func() (RoundOperationResult, error) {
		s.logger.Info("Processing round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("reminder_type", payload.ReminderType),
		)

		// Filter participants who have accepted or are tentative
		var userIDs []sharedtypes.DiscordID
		// Get participants from DB
		participants, err := s.RoundDB.GetParticipants(ctx, payload.RoundID)
		if err != nil {
			s.logger.Error("Failed to get participants for round",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError("GetRound")
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to get round: %w", err)
		}

		for _, p := range participants {
			if p.Response == roundtypes.ResponseAccept || p.Response == roundtypes.ResponseTentative {
				userIDs = append(userIDs, sharedtypes.DiscordID(p.UserID))
			}
		}

		// If no participants to notify, log and return
		if len(userIDs) == 0 {
			s.logger.Warn("No participants to notify for reminder",
				attr.RoundID("round_id", payload.RoundID),
			)
			return RoundOperationResult{
				Success: roundevents.RoundReminderProcessedPayload{
					RoundID: payload.RoundID,
				},
			}, nil
		}

		// Create the Discord notification payload
		discordPayload := roundevents.DiscordReminderPayload{
			RoundID:        payload.RoundID,
			RoundTitle:     payload.RoundTitle,
			StartTime:      payload.StartTime,
			Location:       payload.Location,
			UserIDs:        userIDs,
			ReminderType:   payload.ReminderType,
			EventMessageID: payload.EventMessageID,
		}

		s.logger.Info("Round reminder processed",
			attr.RoundID("round_id", payload.RoundID),
			attr.Int("participants", len(userIDs)),
		)

		return RoundOperationResult{
			Success: discordPayload,
		}, nil
	})
}
