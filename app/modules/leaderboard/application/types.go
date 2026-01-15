package leaderboardservice

import (
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type LeaderboardOperationResult struct {
	// Use the actual domain type to avoid conversions
	Leaderboard leaderboardtypes.LeaderboardData

	// Explicit diffs produced by this operation
	TagChanges []TagChange

	// Domain-level error (validation, invariant failure)
	Err error
}

type TagChange struct {
	GuildID sharedtypes.GuildID
	UserID  sharedtypes.DiscordID
	OldTag  *sharedtypes.TagNumber
	NewTag  *sharedtypes.TagNumber
	Reason  sharedtypes.ServiceUpdateSource
}
