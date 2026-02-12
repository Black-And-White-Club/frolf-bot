package leaderboarddomain

import "time"

// ResolvedSeason holds the resolved season context for a round processing operation.
type ResolvedSeason struct {
	SeasonID string
	IsActive bool
}

// SeasonState represents the current season configuration for a guild.
type SeasonState struct {
	GuildID   string
	SeasonID  string
	IsActive  bool
	StartDate *time.Time
	EndDate   *time.Time
}

// ResolveSeasonForRound determines which season a round should be processed under.
//
// Rules:
//   - If rollbackSeasonID is non-empty, use it (recalculation preserves original season).
//   - If an active season exists, use it.
//   - If no active season, return empty ResolvedSeason (points will be skipped).
func ResolveSeasonForRound(rollbackSeasonID string, activeSeason *SeasonState) ResolvedSeason {
	if rollbackSeasonID != "" {
		return ResolvedSeason{
			SeasonID: rollbackSeasonID,
			IsActive: true,
		}
	}

	if activeSeason != nil && activeSeason.IsActive {
		return ResolvedSeason{
			SeasonID: activeSeason.SeasonID,
			IsActive: true,
		}
	}

	// No active season = off-season. Tags still update, points are skipped.
	return ResolvedSeason{}
}

// ShouldAwardPoints determines whether points should be awarded for a round.
// Points are skipped when there is no active season (off-season mode).
func ShouldAwardPoints(season ResolvedSeason) bool {
	return season.SeasonID != ""
}

// ValidateSeasonStart checks whether a new season can be started for a guild.
// Returns an error message if validation fails, empty string if OK.
func ValidateSeasonStart(seasonID, seasonName string) string {
	if seasonID == "" {
		return "season_id is required"
	}
	if seasonName == "" {
		return "season_name is required"
	}
	return ""
}
