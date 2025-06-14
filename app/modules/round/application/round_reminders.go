package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundReminder handles the reminder event when it's triggered from the delayed queue
func (s *RoundService) ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ProcessRoundReminder", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("reminder_type", payload.ReminderType),
		)

		// Filter participants who have accepted or are tentative
		var userIDs []sharedtypes.DiscordID
		// Get participants from DB
		participants, err := s.RoundDB.GetParticipants(ctx, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participants for round",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetParticipants")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
		}

		for _, p := range participants {
			if p.Response == roundtypes.ResponseAccept || p.Response == roundtypes.ResponseTentative {
				userIDs = append(userIDs, sharedtypes.DiscordID(p.UserID))
			}
		}

		// Create the Discord notification payload with filtered participants
		discordPayload := &roundevents.DiscordReminderPayload{
			RoundID:          payload.RoundID,
			RoundTitle:       payload.RoundTitle,
			StartTime:        payload.StartTime,
			Location:         payload.Location,
			UserIDs:          userIDs, // This could be empty
			ReminderType:     payload.ReminderType,
			EventMessageID:   payload.EventMessageID,
			DiscordChannelID: payload.DiscordChannelID,
			DiscordGuildID:   payload.DiscordGuildID,
		}

		// Log the processing result
		if len(userIDs) == 0 {
			s.logger.Warn("No participants to notify for reminder",
				attr.RoundID("round_id", payload.RoundID),
			)
		} else {
			s.logger.InfoContext(ctx, "Round reminder processed",
				attr.RoundID("round_id", payload.RoundID),
				attr.Int("participants", len(userIDs)),
			)
		}

		// Always return the DiscordReminderPayload (handler will decide what to do with it)
		return RoundOperationResult{
			Success: discordPayload,
		}, nil
	})
}
