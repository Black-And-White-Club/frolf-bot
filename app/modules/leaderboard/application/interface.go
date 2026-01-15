package leaderboardservice

import (
	"context"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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
	) (LeaderboardOperationResult, error)

	// TagSwapRequested now uses domain types.
	// It attempts an assignment and allows the TagSwapNeededError to bubble up.
	TagSwapRequested(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		targetTag sharedtypes.TagNumber,
	) (LeaderboardOperationResult, error)

	// --- READS ---

	GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (LeaderboardOperationResult, error)

	// Simplified lookup: returns the tag or an error (e.g., ErrUserTagNotFound)
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error)

	// RoundGetTagByUserID can stay as-is if it needs to return metadata for round events
	RoundGetTagByUserID(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		payload sharedevents.RoundTagLookupRequestedPayloadV1,
	) (LeaderboardOperationResult, error)

	// CheckTagAvailability validates whether a tag can be assigned to a user.
	CheckTagAvailability(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		tagNumber *sharedtypes.TagNumber,
	) (sharedevents.TagAvailabilityCheckResultPayloadV1, *sharedevents.TagAvailabilityCheckFailedPayloadV1, error)

	// --- INFRASTRUCTURE ---
	EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) error
}
