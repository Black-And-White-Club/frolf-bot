package leaderboardservice

import (
	"context"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
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
	) (results.OperationResult, error)

	// TagSwapRequested attempts an assignment and allows TagSwapNeededError to bubble up.
	TagSwapRequested(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		targetTag sharedtypes.TagNumber,
	) (results.OperationResult, error)

	// --- READS ---

	GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error)

	// GetTagByUserID returns the tag for a user or an error.
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error)

	// RoundGetTagByUserID provides tag lookup with round-specific metadata.
	RoundGetTagByUserID(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		payload sharedevents.RoundTagLookupRequestedPayloadV1,
	) (results.OperationResult, error)

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
