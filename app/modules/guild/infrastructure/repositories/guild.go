package guilddb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

type GuildDBImpl struct {
	DB *bun.DB
}

func (db *GuildDBImpl) GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	var config GuildConfig
	err := db.DB.NewSelect().Model(&config).Where("guild_id = ? AND is_active = true", guildID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Config not found, return nil without error
		}
		return nil, err
	}
	return toSharedModel(&config), nil
}

// toSharedModel converts the DB GuildConfig to the shared type
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
		// Add more fields as needed
	}
}

func (db *GuildDBImpl) SaveConfig(ctx context.Context, config *guildtypes.GuildConfig) error {
	dbModel := toDBModel(config)
	_, err := db.DB.NewInsert().Model(dbModel).Exec(ctx)
	return err
}

// toDBModel converts the shared GuildConfig to the DB model
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
		// Add more fields as needed
	}
}

func (db *GuildDBImpl) UpdateConfig(ctx context.Context, guildID sharedtypes.GuildID, updates map[string]interface{}) error {
	// Convert Go field names to snake_case column names for database
	columnUpdates := make(map[string]interface{})
	for k, v := range updates {
		columnName := toSnakeCase(k)
		columnUpdates[columnName] = v
	}
	
	q := db.DB.NewUpdate().Model(&GuildConfig{}).Where("guild_id = ?", guildID)
	for k, v := range columnUpdates {
		q = q.Set(fmt.Sprintf("%s = ?", k), v)
	}
	_, err := q.Exec(ctx)
	return err
}

// toSnakeCase converts a Go field name to snake_case
// e.g., "EventChannelID" -> "event_channel_id", "EditorRoleID" -> "editor_role_id"
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Don't insert underscore if previous character was uppercase (handles acronyms)
			if i > 0 && s[i-1] >= 'a' && s[i-1] <= 'z' {
				result = append(result, '_')
			}
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func (db *GuildDBImpl) DeleteConfig(ctx context.Context, guildID sharedtypes.GuildID) error {
	_, err := db.DB.NewDelete().Model(&GuildConfig{}).Where("guild_id = ?", guildID).Exec(ctx)
	return err
}
