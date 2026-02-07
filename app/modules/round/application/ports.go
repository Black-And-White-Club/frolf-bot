package roundservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// UserIdentity represents the minimal user information needed during scorecard import.
type UserIdentity struct {
	UserID sharedtypes.DiscordID
}

// UserLookup defines the minimal contract for resolving users by normalized UDisc fields.
// Implementations may hit a database directly or call out to another service.
type UserLookup interface {
	FindByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedUsername string) (*UserIdentity, error)
	FindGlobalByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, normalizedUsername string) (*UserIdentity, error)
	FindByNormalizedUDiscDisplayName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedDisplayName string) (*UserIdentity, error)
	FindByPartialUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserIdentity, error)
}
