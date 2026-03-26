package guilddb

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
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

	// ResolveEntitlements resolves current feature access for a guild or club UUID.
	ResolveEntitlements(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error)

	// GetClubIDByDiscordGuildID performs a lightweight lookup to get the Club UUID for a Discord guild ID.
	GetClubIDByDiscordGuildID(ctx context.Context, guildID string) (uuid.UUID, error)

	// UpsertFeatureOverride inserts or updates a feature override and creates an audit record.
	UpsertFeatureOverride(ctx context.Context, db bun.IDB, override *ClubFeatureOverride, audit *ClubFeatureAccessAudit) error

	// DeleteFeatureOverride deletes a feature override and creates an audit record.
	DeleteFeatureOverride(ctx context.Context, db bun.IDB, clubUUID string, featureKey string, audit *ClubFeatureAccessAudit) error

	// ListFeatureAccessAudit retrieves the audit history for a club's feature.
	ListFeatureAccessAudit(ctx context.Context, db bun.IDB, clubUUID string, featureKey string) ([]ClubFeatureAccessAudit, error)

	// InsertOutboxEvent inserts a pending outbox event within the current
	// transaction. Must be called with a bun.Tx so the INSERT is atomic with
	// the business mutation.
	InsertOutboxEvent(ctx context.Context, db bun.IDB, topic string, payload []byte) error

	// PollAndLockOutboxEvents returns up to limit unpublished outbox rows,
	// locking them with SELECT … FOR UPDATE SKIP LOCKED.
	PollAndLockOutboxEvents(ctx context.Context, db bun.IDB, limit int) ([]GuildOutboxEvent, error)

	// MarkOutboxEventPublished sets published_at for the given row.
	MarkOutboxEventPublished(ctx context.Context, db bun.IDB, id string) error
}
