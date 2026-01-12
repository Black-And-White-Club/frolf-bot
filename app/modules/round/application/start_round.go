package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundStart handles the start of a round, updates participant data, updates DB, and notifies Discord.
// Multi-guild: require guildID for all round operations
func (s *RoundService) ProcessRoundStart(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
) (RoundOperationResult, error) {

	return s.serviceWrapper(ctx, "ProcessRoundStart", roundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round start",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		// Fetch the round from DB (DB is the source of truth)
		round, err := s.RoundDB.GetRound(ctx, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round from database",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Ensure we have an event message id to update/notify Discord
		if round.EventMessageID == "" {
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Error:   "round missing event_message_id",
				},
			}, nil
		}

		// Update the round state to "in progress"
		err = s.RoundDB.UpdateRoundState(ctx, guildID, roundID, roundtypes.RoundStateInProgress)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round state to in progress",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRoundState")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Convert []roundtypes.Participant to []roundevents.RoundParticipantV1
		participants := make([]roundevents.RoundParticipantV1, len(round.Participants))
		for i, p := range round.Participants {
			participants[i] = roundevents.RoundParticipantV1{
				UserID:    sharedtypes.DiscordID(p.UserID),
				TagNumber: p.TagNumber,
				Response:  roundtypes.Response(p.Response),
				Score:     p.Score,
			}
		}

		// Determine Discord channel to use. Prefer guild config if available.
		discordChannelID := ""
		if cfg := s.getGuildConfigForEnrichment(ctx, guildID); cfg != nil && cfg.EventChannelID != "" {
			discordChannelID = cfg.EventChannelID
		}

		// Build the Discord-specific payload from DB values (DB is authoritative)
		return RoundOperationResult{
			Success: &roundevents.DiscordRoundStartPayloadV1{
				GuildID:          guildID,
				RoundID:          roundID,
				Title:            round.Title,
				Location:         round.Location,
				StartTime:        round.StartTime,
				Participants:     participants,
				EventMessageID:   round.EventMessageID,
				DiscordChannelID: discordChannelID,
			},
		}, nil
	})
}
