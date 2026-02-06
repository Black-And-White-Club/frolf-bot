package roundservice

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// GetRound retrieves the round from the database.
// Multi-guild: require guildID for all round operations
func (s *RoundService) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
	return withTelemetry[*roundtypes.Round, error](s, ctx, "GetRound", roundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		s.logger.InfoContext(ctx, "Getting round from database",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		// Pass s.db to repo method
		dbRound, err := s.repo.GetRound(ctx, s.db, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to retrieve round",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return results.FailureResult[*roundtypes.Round, error](err), nil
		}

		s.logger.InfoContext(ctx, "Round retrieved from database",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		return results.SuccessResult[*roundtypes.Round, error](dbRound), nil
	})
}

// GetRoundsForGuild retrieves all rounds for a guild, filtered by active states.
func (s *RoundService) GetRoundsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	ctx, span := s.tracer.Start(ctx, "GetRoundsForGuild")
	defer span.End()

	s.logger.InfoContext(ctx, "Fetching rounds for guild",
		attr.ExtractCorrelationID(ctx),
		attr.String("guild_id", string(guildID)),
	)

	// Get rounds that are not deleted or cancelled (active/completed)
	rounds, err := s.repo.GetRoundsByGuildID(ctx, s.db, guildID,
		roundtypes.RoundStateUpcoming,
		roundtypes.RoundStateInProgress,
		roundtypes.RoundStateFinalized,
	)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to fetch rounds for guild",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(guildID)),
			attr.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch rounds for guild: %w", err)
	}

	s.logger.InfoContext(ctx, "Successfully fetched rounds for guild",
		attr.ExtractCorrelationID(ctx),
		attr.String("guild_id", string(guildID)),
		attr.Int("count", len(rounds)),
	)

	return rounds, nil
}

// GetRoundByDiscordEventID retrieves a round by its Discord native event ID.
// This is used for RSVP resolution when the discord-service's in-memory map is empty.
func (s *RoundService) GetRoundByDiscordEventID(ctx context.Context, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error) {
	ctx, span := s.tracer.Start(ctx, "GetRoundByDiscordEventID")
	defer span.End()

	s.logger.InfoContext(ctx, "Getting round by Discord event ID",
		attr.String("discord_event_id", discordEventID),
		attr.String("guild_id", string(guildID)),
	)

	round, err := s.repo.GetRoundByDiscordEventID(ctx, s.db, guildID, discordEventID)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to retrieve round by Discord event ID",
			attr.String("discord_event_id", discordEventID),
			attr.String("guild_id", string(guildID)),
			attr.Error(err),
		)
		s.metrics.RecordDBOperationError(ctx, "GetRoundByDiscordEventID")
		return nil, fmt.Errorf("failed to get round by discord event ID: %w", err)
	}

	s.logger.InfoContext(ctx, "Round retrieved by Discord event ID",
		attr.String("discord_event_id", discordEventID),
		attr.String("guild_id", string(guildID)),
		attr.RoundID("round_id", round.ID),
	)

	s.metrics.RecordDBOperationSuccess(ctx, "GetRoundByDiscordEventID")
	return round, nil
}
