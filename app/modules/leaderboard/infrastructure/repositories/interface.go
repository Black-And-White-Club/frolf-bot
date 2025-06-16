package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetActiveLeaderboard(ctx context.Context) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, leaderboard *Leaderboard) (int64, error)
	DeactivateLeaderboard(ctx context.Context, leaderboardID int64) error
	UpdateLeaderboard(ctx context.Context, leaderboardData leaderboardtypes.LeaderboardData, UpdateID sharedtypes.RoundID) (*Leaderboard, error)
	SwapTags(ctx context.Context, requestorID, targetID sharedtypes.DiscordID) error
	AssignTag(ctx context.Context, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber, source string, requestUpdateID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) (sharedtypes.RoundID, error)
	GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error)
	CheckTagAvailability(ctx context.Context, tagNumber sharedtypes.TagNumber) (bool, error)
	BatchAssignTags(ctx context.Context, assignments []TagAssignment, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID, userID sharedtypes.DiscordID) error
}
