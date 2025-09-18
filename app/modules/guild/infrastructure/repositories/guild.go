package guilddb

import (
	"context"
	"database/sql"
	"fmt"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

type GuildDBImpl struct {
	DB *bun.DB
}

func (db *GuildDBImpl) GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	var config GuildConfig
	err := db.DB.NewSelect().Model(&config).Where("guild_id = ?", guildID).Scan(ctx)
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
	q := db.DB.NewUpdate().Model(&GuildConfig{}).Where("guild_id = ?", guildID)
	for k, v := range updates {
		q = q.Set(fmt.Sprintf("%s = ?", k), v)
	}
	_, err := q.Exec(ctx)
	return err
}

func (db *GuildDBImpl) DeleteConfig(ctx context.Context, guildID sharedtypes.GuildID) error {
	_, err := db.DB.NewDelete().Model(&GuildConfig{}).Where("guild_id = ?", guildID).Exec(ctx)
	return err
}
