package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// TagAvailabilityResult represents the detailed result of a tag availability check
type TagAvailabilityResult struct {
	Available bool
	Reason    string // Empty if available, otherwise explains why unavailable
}

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetActiveLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (int64, error)
	DeactivateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboardID int64) error
	UpdateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, UpdateID sharedtypes.RoundID) (*Leaderboard, error)
	SwapTags(ctx context.Context, guildID sharedtypes.GuildID, requestorID, targetID sharedtypes.DiscordID) error
	AssignTag(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber, source string, requestUpdateID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) (sharedtypes.RoundID, error)
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error)
	CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (TagAvailabilityResult, error)
	BatchAssignTags(ctx context.Context, guildID sharedtypes.GuildID, assignments []TagAssignment, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID, userID sharedtypes.DiscordID) error
}
