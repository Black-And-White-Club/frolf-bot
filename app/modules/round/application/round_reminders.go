package roundservice

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ProcessRoundReminder handles the reminder event when it's triggered from the delayed queue
// Multi-guild: require guildID for all round operations
func (s *RoundService) ProcessRoundReminder(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (ProcessRoundReminderResult, error) {
	return withTelemetry(s, ctx, "ProcessRoundReminder", req.RoundID, func(ctx context.Context) (ProcessRoundReminderResult, error) {
		s.logger.InfoContext(ctx, "Processing round reminder",
			attr.RoundID("round_id", req.RoundID),
			attr.String("reminder_type", req.ReminderType),
			attr.String("guild_id", string(req.GuildID)),
		)

		// Filter participants who have accepted or are tentative
		var userIDs []sharedtypes.DiscordID
		// Get participants from DB
		participants, err := s.repo.GetParticipants(ctx, nil, req.GuildID, req.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participants for round",
				attr.RoundID("round_id", req.RoundID),
				attr.String("guild_id", string(req.GuildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetParticipants")
			return results.FailureResult[roundtypes.ProcessRoundReminderResult](err), nil
		}

		for _, p := range participants {
			if p.Response == roundtypes.ResponseAccept || p.Response == roundtypes.ResponseTentative {
				userIDs = append(userIDs, sharedtypes.DiscordID(p.UserID))
			}
		}

		// Enrich channel ID if missing before constructing outbound payload
		channelID := req.DiscordChannelID
		if channelID == "" {
			if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil && cfg.EventChannelID != "" {
				channelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Enriched missing DiscordChannelID for reminder from config cache",
					attr.String("channel_id", channelID),
				)
			} else {
				s.logger.WarnContext(ctx, "DiscordChannelID missing for reminder and no config fallback available",
					attr.String("guild_id", string(req.GuildID)),
				)
			}
		}

		// Create the Discord notification payload with filtered participants
		result := &roundtypes.ProcessRoundReminderResult{
			GuildID:          req.GuildID,
			RoundID:          req.RoundID,
			RoundTitle:       req.RoundTitle,
			StartTime:        req.StartTime,
			Location:         req.Location,
			UserIDs:          userIDs, // This could be empty
			ReminderType:     req.ReminderType,
			EventMessageID:   req.EventMessageID,
			DiscordChannelID: channelID,
			DiscordGuildID:   req.DiscordGuildID,
		}

		// Log the processing result
		if len(userIDs) == 0 {
			s.logger.Warn("No participants to notify for reminder",
				attr.RoundID("round_id", req.RoundID),
				attr.String("guild_id", string(req.GuildID)),
			)
		} else {
			s.logger.InfoContext(ctx, "Round reminder processed",
				attr.RoundID("round_id", req.RoundID),
				attr.String("guild_id", string(req.GuildID)),
				attr.Int("participants", len(userIDs)),
			)
		}

		// Always return the DiscordReminderPayload (handler will decide what to do with it)
		return results.SuccessResult[roundtypes.ProcessRoundReminderResult, error](*result), nil
	})
}
