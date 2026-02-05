package clubdb

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Club is the Bun ORM model for the clubs table.
type Club struct {
	bun.BaseModel `bun:"table:clubs,alias:c"`

	UUID           uuid.UUID `bun:"uuid,pk,type:uuid,default:gen_random_uuid()"`
	Name           string    `bun:"name,notnull"`
	IconURL        *string   `bun:"icon_url,nullzero"`
	DiscordGuildID *string   `bun:"discord_guild_id,nullzero,unique"`
	CreatedAt      time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
