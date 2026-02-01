package guilddb

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Repository defines the contract for guild configuration persistence.
// All methods are context-aware for cancellation and timeout propagation.
//
// Error semantics:
//   - ErrNotFound: Record does not exist (GetConfig)
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - Other errors: Infrastructure failures (DB connection, query errors)
type Repository interface {
	// GetConfig retrieves an active guild configuration by ID.
	// Returns ErrNotFound if no active config exists for the guild.
	// All methods accept a Bun DB handle so callers can control transactions
	// and use the same DB connection as the service layer.
	GetConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	GetConfigIncludeDeleted(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)

	// SaveConfig creates or re-activates a guild configuration.
	// Uses UPSERT semantics: inserts if not exists, updates if exists.
	// Re-activation: sets is_active=true, deletion_status='none'.
	SaveConfig(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error

	// UpdateConfig applies partial updates to an active guild configuration.
	// Only non-nil fields in UpdateFields are applied.
	// Returns ErrNoRowsAffected if no active config exists.
	UpdateConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *UpdateFields) error

	// DeleteConfig performs a soft delete with resource state capture.
	// Sets is_active=false, captures resource snapshot for cleanup.
	// Idempotent: no error if already deleted or doesn't exist.
	DeleteConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error
}
