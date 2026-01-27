package leaderboardservice

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Service defines the contract for leaderboard operations.
// All state mutations flow through ExecuteBatchTagAssignment (The Funnel).
type Service interface {
	// --- MUTATIONS (The Funnel) ---

	// ExecuteBatchTagAssignment is the consolidated funnel.
	// All other mutation methods eventually call this internally.
	ExecuteBatchTagAssignment(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		requests []sharedtypes.TagAssignmentRequest,
		updateID sharedtypes.RoundID,
		source sharedtypes.ServiceUpdateSource,
	) (leaderboardtypes.LeaderboardData, error)

	// TagSwapRequested attempts an assignment and returns the updated leaderboard data or an error.
	TagSwapRequested(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		targetTag sharedtypes.TagNumber,
	) (leaderboardtypes.LeaderboardData, error)

	// --- READS ---

	// GetLeaderboard returns the active leaderboard entries as domain types.
	GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardtypes.LeaderboardEntry, error)

	// GetTagByUserID returns the tag for a user or an error.
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error)

	// RoundGetTagByUserID provides tag lookup with round-specific metadata.
	RoundGetTagByUserID(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
	) (sharedtypes.TagNumber, error)

	// CheckTagAvailability validates whether a tag can be assigned to a user.
	CheckTagAvailability(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		tagNumber sharedtypes.TagNumber,
	) (TagAvailabilityResult, error)

	// --- INFRASTRUCTURE ---
	EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) error
}
