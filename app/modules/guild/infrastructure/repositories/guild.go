package guilddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new guild repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

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

// upsertSetColumns defines fields to overwrite on a conflict (SaveConfig).
// Includes deletion_status/resource_state to handle re-activations.
var upsertSetColumns = []string{
	"signup_channel_id", "signup_message_id", "event_channel_id",
	"leaderboard_channel_id", "user_role_id", "editor_role_id",
	"admin_role_id", "signup_emoji", "auto_setup_completed",
	"setup_completed_at", "is_active", "updated_at",
	"deletion_status", "resource_state",
}

// --- READ METHODS ---

// GetConfig retrieves an active guild configuration by ID.
func (r *Impl) GetConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	if db == nil {
		db = r.db
	}

	model := new(GuildConfig)
	err := db.NewSelect().
		Model(model).
		Where("guild_id = ? AND is_active = true", guildID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("guilddb.GetConfig: %w", err)
	}
	return toSharedModel(model), nil
}

// --- WRITE METHODS ---

// SaveConfig creates or re-activates a guild configuration.
func (r *Impl) SaveConfig(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error {
	if db == nil {
		db = r.db
	}

	dbModel := toDBModel(config)
	dbModel.IsActive = true
	dbModel.UpdatedAt = time.Now().UTC()
	dbModel.DeletionStatus = "none"

	q := db.NewInsert().
		Model(dbModel).
		On("CONFLICT (guild_id) DO UPDATE")

	// Reuse existing upsert columns logic
	for _, col := range upsertSetColumns {
		q = q.Set("? = EXCLUDED.?", bun.Ident(col), bun.Ident(col))
	}

	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("guilddb.SaveConfig: %w", err)
	}
	return nil
}

// UpdateConfig applies partial updates to an active guild configuration.
func (r *Impl) UpdateConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *UpdateFields) error {
	if updates == nil || updates.IsEmpty() {
		return nil
	}
	if db == nil {
		db = r.db
	}

	q := db.NewUpdate().
		Table("guild_configs").
		Where("guild_id = ? AND is_active = true", guildID)

	// Apply partial updates
	if updates.SignupChannelID != nil {
		q = q.Set("signup_channel_id = ?", *updates.SignupChannelID)
	}
	if updates.SignupMessageID != nil {
		q = q.Set("signup_message_id = ?", *updates.SignupMessageID)
	}
	if updates.EventChannelID != nil {
		q = q.Set("event_channel_id = ?", *updates.EventChannelID)
	}
	if updates.LeaderboardChannelID != nil {
		q = q.Set("leaderboard_channel_id = ?", *updates.LeaderboardChannelID)
	}
	if updates.UserRoleID != nil {
		q = q.Set("user_role_id = ?", *updates.UserRoleID)
	}
	if updates.EditorRoleID != nil {
		q = q.Set("editor_role_id = ?", *updates.EditorRoleID)
	}
	if updates.AdminRoleID != nil {
		q = q.Set("admin_role_id = ?", *updates.AdminRoleID)
	}
	if updates.SignupEmoji != nil {
		q = q.Set("signup_emoji = ?", *updates.SignupEmoji)
	}
	if updates.AutoSetupCompleted != nil {
		q = q.Set("auto_setup_completed = ?", *updates.AutoSetupCompleted)
	}
	if updates.SetupCompletedAt != nil {
		q = q.Set("setup_completed_at = ?", unixNanoToTime(*updates.SetupCompletedAt))
	}

	q = q.Set("updated_at = ?", time.Now().UTC())

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("guilddb.UpdateConfig: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// DeleteConfig performs a soft delete.
// Note: Transaction handling is removed here as it should be managed by the Service.
func (r *Impl) DeleteConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error {
	if db == nil {
		db = r.db
	}

	model := new(GuildConfig)
	if err := db.NewSelect().Model(model).Where("guild_id = ? AND is_active = true", guildID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("guilddb.DeleteConfig select: %w", err)
	}

	// Capture resource state snapshot for cleanup
	rs := &ResourceState{
		SignupChannelID:      model.SignupChannelID,
		SignupMessageID:      model.SignupMessageID,
		EventChannelID:       model.EventChannelID,
		LeaderboardChannelID: model.LeaderboardChannelID,
		UserRoleID:           model.UserRoleID,
		EditorRoleID:         model.EditorRoleID,
		AdminRoleID:          model.AdminRoleID,
		Results:              make(map[string]DeletionResult),
	}

	_, err := db.NewUpdate().
		Table("guild_configs").
		Where("guild_id = ?", guildID).
		Set("resource_state = ?", rs).
		Set("deletion_status = 'pending'").
		Set("is_active = false").
		Set("updated_at = ?", time.Now().UTC()).
		Set("signup_channel_id = NULL, signup_message_id = NULL, event_channel_id = NULL, leaderboard_channel_id = NULL").
		Set("user_role_id = NULL, editor_role_id = NULL, admin_role_id = NULL").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("guilddb.DeleteConfig update: %w", err)
	}
	return nil
}

// =============================================================================
// Model Conversion Helpers
// =============================================================================

// toSharedModel converts the DB model to the shared domain type.
func toSharedModel(cfg *GuildConfig) *guildtypes.GuildConfig {
	if cfg == nil {
		return nil
	}
	return &guildtypes.GuildConfig{
		GuildID:              cfg.GuildID,
		SignupChannelID:      cfg.SignupChannelID,
		SignupMessageID:      cfg.SignupMessageID,
		EventChannelID:       cfg.EventChannelID,
		LeaderboardChannelID: cfg.LeaderboardChannelID,
		UserRoleID:           cfg.UserRoleID,
		EditorRoleID:         cfg.EditorRoleID,
		AdminRoleID:          cfg.AdminRoleID,
		SignupEmoji:          cfg.SignupEmoji,
		AutoSetupCompleted:   cfg.AutoSetupCompleted,
		SetupCompletedAt:     cfg.SetupCompletedAt,
		ResourceState:        toSharedResourceState(cfg.ResourceState),
	}
}

// toDBModel converts the shared domain type to the DB model.
func toDBModel(cfg *guildtypes.GuildConfig) *GuildConfig {
	if cfg == nil {
		return nil
	}
	return &GuildConfig{
		GuildID:              cfg.GuildID,
		SignupChannelID:      cfg.SignupChannelID,
		SignupMessageID:      cfg.SignupMessageID,
		EventChannelID:       cfg.EventChannelID,
		LeaderboardChannelID: cfg.LeaderboardChannelID,
		UserRoleID:           cfg.UserRoleID,
		EditorRoleID:         cfg.EditorRoleID,
		AdminRoleID:          cfg.AdminRoleID,
		SignupEmoji:          cfg.SignupEmoji,
		AutoSetupCompleted:   cfg.AutoSetupCompleted,
		SetupCompletedAt:     cfg.SetupCompletedAt,
		ResourceState:        toDBResourceState(&cfg.ResourceState),
	}
}

// toSharedResourceState converts DB ResourceState to shared type.
func toSharedResourceState(rs *ResourceState) guildtypes.ResourceState {
	if rs == nil {
		return guildtypes.ResourceState{}
	}
	results := make(map[string]guildtypes.DeletionResult, len(rs.Results))
	for k, v := range rs.Results {
		results[k] = guildtypes.DeletionResult{
			Status:    v.Status,
			Error:     v.Error,
			DeletedAt: v.DeletedAt,
		}
	}
	return guildtypes.ResourceState{
		SignupChannelID:      rs.SignupChannelID,
		SignupMessageID:      rs.SignupMessageID,
		EventChannelID:       rs.EventChannelID,
		LeaderboardChannelID: rs.LeaderboardChannelID,
		UserRoleID:           rs.UserRoleID,
		EditorRoleID:         rs.EditorRoleID,
		AdminRoleID:          rs.AdminRoleID,
		Results:              results,
	}
}

// toDBResourceState converts shared ResourceState to DB type.
func toDBResourceState(rs *guildtypes.ResourceState) *ResourceState {
	if rs == nil || rs.IsEmpty() {
		return nil
	}
	results := make(map[string]DeletionResult, len(rs.Results))
	for k, v := range rs.Results {
		results[k] = DeletionResult{
			Status:    v.Status,
			Error:     v.Error,
			DeletedAt: v.DeletedAt,
		}
	}
	return &ResourceState{
		SignupChannelID:      rs.SignupChannelID,
		SignupMessageID:      rs.SignupMessageID,
		EventChannelID:       rs.EventChannelID,
		LeaderboardChannelID: rs.LeaderboardChannelID,
		UserRoleID:           rs.UserRoleID,
		EditorRoleID:         rs.EditorRoleID,
		AdminRoleID:          rs.AdminRoleID,
		Results:              results,
	}
}

// unixNanoToTime converts a unix nano timestamp to *time.Time.
func unixNanoToTime(nano int64) *time.Time {
	if nano == 0 {
		return nil
	}
	t := time.Unix(0, nano).UTC()
	return &t
}
