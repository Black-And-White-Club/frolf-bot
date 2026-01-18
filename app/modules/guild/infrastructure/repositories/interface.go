package guilddb

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// UpdateFields represents the updateable fields of a guild config.
// Pointer fields distinguish "not provided" (nil) from "set to zero value".
// This enables clean partial updates without full object replacement.
type UpdateFields struct {
	SignupChannelID      *string
	SignupMessageID      *string
	EventChannelID       *string
	LeaderboardChannelID *string
	UserRoleID           *string
	EditorRoleID         *string
	AdminRoleID          *string
	SignupEmoji          *string
	AutoSetupCompleted   *bool
	SetupCompletedAt     *int64 // Unix nano timestamp
}

// IsEmpty reports whether any fields are set for update.
func (u *UpdateFields) IsEmpty() bool {
	if u == nil {
		return true
	}
	return u.SignupChannelID == nil &&
		u.SignupMessageID == nil &&
		u.EventChannelID == nil &&
		u.LeaderboardChannelID == nil &&
		u.UserRoleID == nil &&
		u.EditorRoleID == nil &&
		u.AdminRoleID == nil &&
		u.SignupEmoji == nil &&
		u.AutoSetupCompleted == nil &&
		u.SetupCompletedAt == nil
}

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
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)

	// SaveConfig creates or re-activates a guild configuration.
	// Uses UPSERT semantics: inserts if not exists, updates if exists.
	// Re-activation: sets is_active=true, deletion_status='none'.
	SaveConfig(ctx context.Context, config *guildtypes.GuildConfig) error

	// UpdateConfig applies partial updates to an active guild configuration.
	// Only non-nil fields in UpdateFields are applied.
	// Returns ErrNoRowsAffected if no active config exists.
	UpdateConfig(ctx context.Context, guildID sharedtypes.GuildID, updates *UpdateFields) error

	// DeleteConfig performs a soft delete with resource state capture.
	// Sets is_active=false, captures resource snapshot for cleanup.
	// Idempotent: no error if already deleted or doesn't exist.
	DeleteConfig(ctx context.Context, guildID sharedtypes.GuildID) error
}
