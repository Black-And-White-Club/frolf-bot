package guilddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

type GuildDBImpl struct {
	// Supports both *bun.DB and *bun.Tx
	DB bun.IDB
}

// allowedUpdateColumns defines which columns can be updated via UpdateConfig.
// Keys are in snake_case to match database columns.
var allowedUpdateColumns = map[string]struct{}{
	"signup_channel_id":      {},
	"signup_message_id":      {},
	"event_channel_id":       {},
	"leaderboard_channel_id": {},
	"user_role_id":           {},
	"editor_role_id":         {},
	"admin_role_id":          {},
	"signup_emoji":           {},
	"auto_setup_completed":   {},
	"setup_completed_at":     {},
	// Future fields commented out until supported
	// "subscription_tier":           {},
	// "max_concurrent_rounds":       {},
	// "max_participants_per_round":  {},
	// "commands_per_minute":         {},
	// "rounds_per_day":              {},
	// "custom_leaderboards_enabled": {},
}

// upsertColumns defines the columns that are updated when a conflict occurs during SaveConfig.
// This ensures that the configuration is always fully synchronized.
var upsertColumns = []string{
	"signup_channel_id",
	"signup_message_id",
	"event_channel_id",
	"leaderboard_channel_id",
	"user_role_id",
	"editor_role_id",
	"admin_role_id",
	"signup_emoji",
	"auto_setup_completed",
	"setup_completed_at",
	// Future fields commented out until supported
	// "subscription_tier",
	// "max_concurrent_rounds",
	// "max_participants_per_round",
	// "commands_per_minute",
	// "rounds_per_day",
	// "custom_leaderboards_enabled",
	"is_active",  // Re-activate if it was soft-deleted
	"updated_at", // Always refresh timestamp
}

//
// READ

func (db *GuildDBImpl) GetConfig(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) (*guildtypes.GuildConfig, error) {

	var model GuildConfig

	err := db.DB.NewSelect().
		Model(&model).
		Where("guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get guild config guild_id=%s: %w", guildID, err)
	}

	// Respect soft delete, but keep semantic distinction internally
	if !model.IsActive {
		return nil, nil
	}

	return toSharedModel(&model), nil
}

// SaveConfig creates or updates (upsert) the guild configuration.
func (db *GuildDBImpl) SaveConfig(
	ctx context.Context,
	config *guildtypes.GuildConfig,
) error {
	dbModel := toDBModel(config)
	dbModel.IsActive = true
	dbModel.UpdatedAt = time.Now().UTC()

	q := db.DB.NewInsert().
		Model(dbModel).
		On("CONFLICT (guild_id) DO UPDATE")

	// Explicitly set columns to update on conflict.
	for _, col := range upsertColumns {
		q = q.Set("? = EXCLUDED.?", bun.Ident(col), bun.Ident(col))
	}

	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("save guild config guild_id=%s: %w", config.GuildID, err)
	}

	return nil
}

// UpdateConfig performs a partial update of the guild configuration.
// It accepts a map of fields to update, which are sanitized and mapped to DB columns.
func (db *GuildDBImpl) UpdateConfig(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	updates map[string]any,
) error {
	if len(updates) == 0 {
		return nil
	}

	sanitized, err := sanitizeAndMapUpdates(updates)
	if err != nil {
		return fmt.Errorf("sanitize updates: %w", err)
	}

	if len(sanitized) == 0 {
		return nil
	}

	// Always update the updated_at timestamp.
	sanitized["updated_at"] = time.Now().UTC()

	q := db.DB.NewUpdate().
		Table("guild_configs").
		Where("guild_id = ? AND is_active = true", guildID)

	for col, val := range sanitized {
		q = q.SetColumn(col, "?", val)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("update guild config guild_id=%s: %w", guildID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		// This could mean the guild doesn't exist or is inactive.
		// We treat it as ErrNoRows to be consistent.
		return sql.ErrNoRows
	}

	return nil
}

// DeleteConfig soft-deletes the guild configuration.
func (db *GuildDBImpl) DeleteConfig(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) error {
	// Snapshot current resource IDs into `resource_state` and nullify explicit ID
	// columns in a single transaction. Do not perform any external IO here.
	return db.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var model GuildConfig

		err := tx.NewSelect().
			Model(&model).
			Where("guild_id = ? AND is_active = ?", guildID, true).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Nothing to do; idempotent.
				return nil
			}
			return fmt.Errorf("get guild config for delete guild_id=%s: %w", guildID, err)
		}

		// Build resource snapshot
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

		// Prepare update: set resource_state JSONB, nullify explicit columns, mark inactive.
		q := tx.NewUpdate().
			Table("guild_configs").
			Where("guild_id = ? AND is_active = ?", guildID, true)

		q = q.Set("resource_state = ?", rs)
		// Nullify explicit ID columns so the rest of the app no longer treats them as live.
		q = q.Set("signup_channel_id = NULL")
		q = q.Set("signup_message_id = NULL")
		q = q.Set("event_channel_id = NULL")
		q = q.Set("leaderboard_channel_id = NULL")
		q = q.Set("user_role_id = NULL")
		q = q.Set("editor_role_id = NULL")
		q = q.Set("admin_role_id = NULL")

		q = q.Set("is_active = ?", false)
		q = q.Set("updated_at = ?", time.Now().UTC())

		if _, err := q.Exec(ctx); err != nil {
			return fmt.Errorf("delete guild config guild_id=%s: %w", guildID, err)
		}

		return nil
	})
}

//
// Helpers
//

// sanitizeAndMapUpdates validates and converts update map keys to snake_case.
func sanitizeAndMapUpdates(updates map[string]any) (map[string]any, error) {
	clean := make(map[string]any, len(updates))

	for k, v := range updates {
		col := toSnakeCase(k)

		// Prevent updating protected fields explicitly via this method.
		switch col {
		case "guild_id", "created_at", "updated_at", "is_active":
			return nil, fmt.Errorf("field %q cannot be updated manually", k)
		}

		if _, ok := allowedUpdateColumns[col]; !ok {
			return nil, fmt.Errorf("unknown or disallowed update field %q", k)
		}

		clean[col] = v
	}

	return clean, nil
}

// toSharedModel converts the DB model to the domain model.
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
		ResourceState:        mapResourceStateToShared(cfg.ResourceState),
	}
}

// toDBModel converts the domain model to the DB model.
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
		ResourceState:        mapResourceStateToDB(&cfg.ResourceState),
	}
}

// mapResourceStateToShared converts DB ResourceState to shared type
func mapResourceStateToShared(rs *ResourceState) guildtypes.ResourceState {
	if rs == nil {
		return guildtypes.ResourceState{}
	}
	// Convert DeletionResult map
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

// mapResourceStateToDB converts shared ResourceState to DB ResourceState pointer
func mapResourceStateToDB(rs *guildtypes.ResourceState) *ResourceState {
	// Treat an empty or nil ResourceState as nil to avoid storing empty JSONB.
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

// toSnakeCase converts "CamelCase" or "mixedCase" to "snake_case".
func toSnakeCase(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 5) // Preallocate with some buffer (e.g. for underscores)

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
				b.WriteRune('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}
