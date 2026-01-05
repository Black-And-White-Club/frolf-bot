package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// Service handles leaderboard logic.
type Service interface {
	// Single unified tag assignment method
	ProcessTagAssignments(ctx context.Context, guildID sharedtypes.GuildID, source interface{}, requests []sharedtypes.TagAssignmentRequest, requestingUserID *sharedtypes.DiscordID, operationID uuid.UUID, batchID uuid.UUID) (LeaderboardOperationResult, error)

	// Tag Swapping
	TagSwapRequested(ctx context.Context, guildID sharedtypes.GuildID, payload leaderboardevents.TagSwapRequestedPayloadV1) (LeaderboardOperationResult, error)

	// Other Operations
	GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (LeaderboardOperationResult, error)
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (LeaderboardOperationResult, error)
	RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, payload sharedevents.RoundTagLookupRequestedPayloadV1) (LeaderboardOperationResult, error)
	CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, payload leaderboardevents.TagAvailabilityCheckRequestedPayloadV1) (*leaderboardevents.TagAvailabilityCheckResultPayloadV1, *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, error)
	// Ensure an active leaderboard exists for the guild
	EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) error
}
