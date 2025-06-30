package guilddb

import (
	"context"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

type GuildDBImpl struct {
	DB *bun.DB
}

func (db *GuildDBImpl) GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*GuildConfig, error) {
	var config GuildConfig
	err := db.DB.NewSelect().Model(&config).Where("guild_id = ?", guildID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (db *GuildDBImpl) SaveConfig(ctx context.Context, config *GuildConfig) error {
	_, err := db.DB.NewInsert().Model(config).Exec(ctx)
	return err
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
