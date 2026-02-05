package userdb

import (
	"fmt"
	"strconv"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User represents a global user identity (source of truth).
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64                  `bun:"id,pk,autoincrement" json:"id"`
	UUID          uuid.UUID              `bun:"uuid,unique,notnull,default:gen_random_uuid()" json:"uuid"`
	UserID        *sharedtypes.DiscordID `bun:"user_id,unique" json:"user_id"`
	UDiscUsername *string                `bun:"udisc_username,nullzero" json:"udisc_username,omitempty"` // @username
	UDiscName     *string                `bun:"udisc_name,nullzero" json:"udisc_name,omitempty"`         // Name shown on casual rounds
	CreatedAt     time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// ORM relationships
	Memberships []*GuildMembership `bun:"rel:has-many,join:user_id=user_id" json:"-"`

	// Profile fields
	DisplayName      *string    `bun:"display_name,nullzero" json:"display_name,omitempty"`
	AvatarHash       *string    `bun:"avatar_hash,nullzero" json:"avatar_hash,omitempty"`
	ProfileUpdatedAt *time.Time `bun:"profile_updated_at,nullzero" json:"profile_updated_at,omitempty"`
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

// ClubMembership represents a user's membership in a specific club.
type ClubMembership struct {
	bun.BaseModel `bun:"table:club_memberships,alias:cm"`
	ID            uuid.UUID                `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	UserUUID      uuid.UUID                `bun:"user_uuid,notnull,type:uuid" json:"user_uuid"`
	ClubUUID      uuid.UUID                `bun:"club_uuid,notnull,type:uuid" json:"club_uuid"`
	DisplayName   *string                  `bun:"display_name" json:"display_name"`
	AvatarURL     *string                  `bun:"avatar_url" json:"avatar_url"`
	Role          sharedtypes.UserRoleEnum `bun:"role,notnull,default:'user'" json:"role"`
	Source        string                   `bun:"source,notnull,default:'discord'" json:"source"`
	ExternalID    *string                  `bun:"external_id" json:"external_id"`
	JoinedAt      time.Time                `bun:"joined_at,notnull,default:current_timestamp" json:"joined_at"`
	UpdatedAt     time.Time                `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	SyncedAt      *time.Time               `bun:"synced_at" json:"synced_at"` // When display_name was last synced from Discord

	// ORM relationships
	User *User `bun:"rel:belongs-to,join:user_uuid=uuid" json:"-"`
}

// RefreshToken represents a long-lived session token.
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens,alias:rt"`

	Hash        string     `bun:"hash,pk"`
	UserUUID    uuid.UUID  `bun:"user_uuid,notnull"`
	TokenFamily string     `bun:"token_family,notnull"` // Detect token reuse
	ExpiresAt   time.Time  `bun:"expires_at,notnull"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	LastUsedAt  *time.Time `bun:"last_used_at"`
	Revoked     bool       `bun:"revoked,notnull,default:false"`
	RevokedAt   *time.Time `bun:"revoked_at"`
	RevokedBy   *string    `bun:"revoked_by"` // 'user' | 'admin' | 'security'
	IPAddress   *string    `bun:"ip_address,nullzero"`
	UserAgent   *string    `bun:"user_agent,nullzero"`
}

// MagicLink represents a one-time login token.
type MagicLink struct {
	bun.BaseModel `bun:"table:magic_links,alias:ml"`

	Token     string     `bun:"token,pk"`
	UserUUID  uuid.UUID  `bun:"user_uuid,notnull"`
	GuildID   string     `bun:"guild_id,notnull"`
	Role      string     `bun:"role,notnull"`
	ExpiresAt time.Time  `bun:"expires_at,notnull"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	Used      bool       `bun:"used,notnull,default:false"`
	UsedAt    *time.Time `bun:"used_at"`
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
	if u.UserID == nil {
		return ""
	}
	return *u.UserID
}

// AvatarURL returns the Discord CDN URL for the user's avatar
func (u *User) AvatarURL(size int) string {
	if u.AvatarHash != nil && *u.AvatarHash != "" {
		ext := "png"
		// Animated avatars start with "a_"
		if len(*u.AvatarHash) > 2 && (*u.AvatarHash)[:2] == "a_" {
			ext = "gif"
		}
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s?size=%d",
			u.GetUserID(), *u.AvatarHash, ext, size)
	}
	// Default avatar based on user ID
	userIDInt, _ := strconv.ParseUint(string(u.GetUserID()), 10, 64)
	index := (userIDInt >> 22) % 6
	return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)
}

// GetDisplayName returns display name or a fallback
func (u *User) GetDisplayName() string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	// Fallback to last 6 chars of Discord ID
	id := string(u.GetUserID())
	if len(id) > 6 {
		return "User ..." + id[len(id)-6:]
	}
	if id != "" {
		return "User " + id
	}
	return "User " + u.UUID.String()[:8]
}
