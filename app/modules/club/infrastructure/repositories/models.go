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

// ClubInvite is the Bun ORM model for the club_invites table.
type ClubInvite struct {
	bun.BaseModel `bun:"table:club_invites,alias:ci"`

	UUID      uuid.UUID  `bun:"uuid,pk,type:uuid,default:gen_random_uuid()"`
	ClubUUID  uuid.UUID  `bun:"club_uuid,notnull,type:uuid"`
	CreatedBy uuid.UUID  `bun:"created_by,notnull,type:uuid"`
	Code      string     `bun:"code,notnull"`
	Role      string     `bun:"role,notnull,default:'player'"`
	MaxUses   *int       `bun:"max_uses"`
	UseCount  int        `bun:"use_count,notnull,default:0"`
	ExpiresAt *time.Time `bun:"expires_at"`
	Revoked   bool       `bun:"revoked,notnull,default:false"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp"`
}
