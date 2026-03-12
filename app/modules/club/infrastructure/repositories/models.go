package clubdb

import (
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
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

// ClubChallenge is the Bun ORM model for the club_challenges table.
type ClubChallenge struct {
	bun.BaseModel `bun:"table:club_challenges,alias:cc"`

	UUID                  uuid.UUID                 `bun:"uuid,pk,type:uuid,default:gen_random_uuid()"`
	ClubUUID              uuid.UUID                 `bun:"club_uuid,notnull,type:uuid"`
	ChallengerUserUUID    uuid.UUID                 `bun:"challenger_user_uuid,notnull,type:uuid"`
	DefenderUserUUID      uuid.UUID                 `bun:"defender_user_uuid,notnull,type:uuid"`
	Status                clubtypes.ChallengeStatus `bun:"status,notnull"`
	OriginalChallengerTag *int                      `bun:"original_challenger_tag"`
	OriginalDefenderTag   *int                      `bun:"original_defender_tag"`
	OpenedAt              time.Time                 `bun:"opened_at,notnull,default:current_timestamp"`
	OpenExpiresAt         *time.Time                `bun:"open_expires_at"`
	AcceptedAt            *time.Time                `bun:"accepted_at"`
	AcceptedExpiresAt     *time.Time                `bun:"accepted_expires_at"`
	CompletedAt           *time.Time                `bun:"completed_at"`
	HiddenAt              *time.Time                `bun:"hidden_at"`
	HiddenByUserUUID      *uuid.UUID                `bun:"hidden_by_user_uuid,type:uuid"`
	DiscordGuildID        *string                   `bun:"discord_guild_id"`
	DiscordChannelID      *string                   `bun:"discord_channel_id"`
	DiscordMessageID      *string                   `bun:"discord_message_id"`
	CreatedAt             time.Time                 `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt             time.Time                 `bun:"updated_at,notnull,default:current_timestamp"`
}

// ClubChallengeRoundLink tracks round associations for challenges.
type ClubChallengeRoundLink struct {
	bun.BaseModel `bun:"table:club_challenge_round_links,alias:ccrl"`

	UUID               uuid.UUID  `bun:"uuid,pk,type:uuid,default:gen_random_uuid()"`
	ChallengeUUID      uuid.UUID  `bun:"challenge_uuid,notnull,type:uuid"`
	RoundID            uuid.UUID  `bun:"round_id,notnull,type:uuid"`
	LinkedByUserUUID   *uuid.UUID `bun:"linked_by_user_uuid,type:uuid"`
	UnlinkedByUserUUID *uuid.UUID `bun:"unlinked_by_user_uuid,type:uuid"`
	LinkedAt           time.Time  `bun:"linked_at,notnull,default:current_timestamp"`
	UnlinkedAt         *time.Time `bun:"unlinked_at"`
}
