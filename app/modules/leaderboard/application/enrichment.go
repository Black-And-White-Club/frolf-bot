package leaderboardservice

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// enrichWithSeasonData enriches leaderboard entries with season statistics (points, rounds played).
// It modifies the entries slice in place.
func (s *LeaderboardService) enrichWithSeasonData(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	seasonID string,
	entries []leaderboardtypes.LeaderboardEntry,
) error {
	if len(entries) == 0 {
		return nil
	}

	userIDs := make([]sharedtypes.DiscordID, len(entries))
	for i, e := range entries {
		userIDs[i] = e.UserID
	}

	standings, err := s.repo.GetSeasonStandings(ctx, db, string(guildID), seasonID, userIDs)
	if err != nil {
		// Log error but don't fail the request; return base entries without enrichment.
		s.logger.ErrorContext(ctx, "failed to enrich normalized leaderboard with season standings", attr.Error(err))
		return nil
	}

	for i := range entries {
		if st, ok := standings[entries[i].UserID]; ok {
			entries[i].TotalPoints = st.TotalPoints
			entries[i].RoundsPlayed = st.RoundsPlayed
		}
	}
	return nil
}
