package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundReminder handles the reminder event when it's triggered from the delayed queue
// Multi-guild: require guildID for all round operations
func (s *RoundService) ProcessRoundReminder(ctx context.Context, payload roundevents.DiscordReminderPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ProcessRoundReminder", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("reminder_type", payload.ReminderType),
			attr.String("guild_id", string(payload.GuildID)),
		)

		// Filter participants who have accepted or are tentative
		var userIDs []sharedtypes.DiscordID
		// Get participants from DB
		participants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participants for round",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetParticipants")
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
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

		// Enrich channel ID if missing before constructing outbound payload
		channelID := payload.DiscordChannelID
		if channelID == "" {
			if cfg := s.getGuildConfigForEnrichment(ctx, payload.GuildID); cfg != nil && cfg.EventChannelID != "" {
				channelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Enriched missing DiscordChannelID for reminder from config cache",
					attr.String("channel_id", channelID),
				)
			} else {
				s.logger.WarnContext(ctx, "DiscordChannelID missing for reminder and no config fallback available",
					attr.String("guild_id", string(payload.GuildID)),
				)
			}
		}

		// Create the Discord notification payload with filtered participants
		discordPayload := &roundevents.DiscordReminderPayloadV1{
			GuildID:          payload.GuildID,
			RoundID:          payload.RoundID,
			RoundTitle:       payload.RoundTitle,
			StartTime:        payload.StartTime,
			Location:         payload.Location,
			UserIDs:          userIDs, // This could be empty
			ReminderType:     payload.ReminderType,
			EventMessageID:   payload.EventMessageID,
			DiscordChannelID: channelID,
			DiscordGuildID:   payload.DiscordGuildID,
		}

		// Log the processing result
		if len(userIDs) == 0 {
			s.logger.Warn("No participants to notify for reminder",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
			)
		} else {
			s.logger.InfoContext(ctx, "Round reminder processed",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.Int("participants", len(userIDs)),
			)
		}

		// Always return the DiscordReminderPayload (handler will decide what to do with it)
		return results.OperationResult{
			Success: discordPayload,
		}, nil
	})
}
