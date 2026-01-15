package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// TagAvailabilityResult represents the detailed result of a tag availability check
type TagAvailabilityResult struct {
	Available bool
	Reason    string
}

// LeaderboardDB represents the interface for interacting with the leaderboard database.
// Updated for 2026 Best Practices: Transactional Awareness via bun.IDB
type LeaderboardDB interface {
	// Read Methods
	GetActiveLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (*Leaderboard, error)
	GetActiveLeaderboardIDB(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID) (*Leaderboard, error)
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error)
	CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (TagAvailabilityResult, error)

	// Write Methods (Atomic & Batch-Oriented)
	// We pass bun.IDB so the service can control the transaction.
	UpdateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (*Leaderboard, error)
	DeactivateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error
}
