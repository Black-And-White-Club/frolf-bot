package userdb

import (
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// User represents a global user identity (source of truth).
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64                 `bun:"id,pk,autoincrement" json:"id"`
	UserID        sharedtypes.DiscordID `bun:"user_id,unique,notnull" json:"user_id"`
	UDiscUsername *string               `bun:"udisc_username,nullzero" json:"udisc_username,omitempty"` // @username
	UDiscName     *string               `bun:"udisc_name,nullzero" json:"udisc_name,omitempty"`         // Name shown on casual rounds
	CreatedAt     time.Time             `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time             `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// ORM relationships
	Memberships []*GuildMembership `bun:"rel:has-many,join:user_id=user_id" json:"-"`
}

// GuildMembership represents a user's membership in a specific guild.
type GuildMembership struct {
	bun.BaseModel `bun:"table:guild_memberships,alias:gm"`
	ID            int64                    `bun:"id,pk,autoincrement" json:"id"`
	UserID        sharedtypes.DiscordID    `bun:"user_id,notnull" json:"user_id"`
	GuildID       sharedtypes.GuildID      `bun:"guild_id,notnull" json:"guild_id"`
	Role          sharedtypes.UserRoleEnum `bun:"role,notnull,default:'User'" json:"role"`
	JoinedAt      time.Time                `bun:"joined_at,notnull,default:current_timestamp" json:"joined_at"`

	// ORM relationships
	// User is a hard FK; Guild is a logical link (no constraint) for module order-independence
	User *User `bun:"rel:belongs-to,join:user_id=user_id" json:"-"`
}

// UserWithMembership combines user identity with guild-specific data.
// Used for queries that need both global and guild context.
type UserWithMembership struct {
	*User    `bun:",extend"`
	Role     sharedtypes.UserRoleEnum `bun:"role"`
	JoinedAt time.Time                `bun:"joined_at"`
}

// Add these methods to your User struct
func (u *User) GetID() int64 {
	return u.ID
}

// func (u *User) GetName() string {
// 	return u.Name
// }

func (u *User) GetUserID() sharedtypes.DiscordID {
	return u.UserID
}
