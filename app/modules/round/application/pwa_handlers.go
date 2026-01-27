package roundservice

import (
	"context"
	"fmt"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// GetRoundsForGuild retrieves all rounds for a guild, filtered by active states
func (s *RoundService) GetRoundsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	ctx, span := s.tracer.Start(ctx, "GetRoundsForGuild")
	defer span.End()

	s.logger.InfoContext(ctx, "Fetching rounds for guild",
		attr.ExtractCorrelationID(ctx),
		attr.String("guild_id", string(guildID)),
	)

	// Get rounds that are not deleted
	rounds, err := s.repo.GetRoundsByGuildID(ctx, guildID,
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
