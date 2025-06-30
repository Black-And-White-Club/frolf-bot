package guilddb

import (
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// GuildConfig represents a Discord server's configuration and metadata.
type GuildConfig struct {
	bun.BaseModel        `bun:"table:guild_configs,alias:g"`
	GuildID              sharedtypes.GuildID `bun:"guild_id,pk,notnull,type:varchar(20)"`
	CreatedAt            time.Time           `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt            time.Time           `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	IsActive             bool                `bun:"is_active,notnull,default:true"`
	SignupChannelID      string              `bun:"signup_channel_id,nullzero,type:varchar(20)"`
	SignupMessageID      string              `bun:"signup_message_id,nullzero,type:varchar(20)"`
	EventChannelID       string              `bun:"event_channel_id,nullzero,type:varchar(20)"`
	LeaderboardChannelID string              `bun:"leaderboard_channel_id,nullzero,type:varchar(20)"`
	UserRoleID           string              `bun:"user_role_id,nullzero,type:varchar(20)"`
	EditorRoleID         string              `bun:"editor_role_id,nullzero,type:varchar(20)"`
	AdminRoleID          string              `bun:"admin_role_id,nullzero,type:varchar(20)"`
	SignupEmoji          string              `bun:"signup_emoji,notnull,default:'🐍',type:varchar(10)"`
	AutoSetupCompleted   bool                `bun:"auto_setup_completed,notnull,default:false"`
	SetupCompletedAt     *time.Time          `bun:"setup_completed_at,nullzero"`
	// Optional/future fields:
	SubscriptionTier          string     `bun:"subscription_tier,type:varchar(20)"`
	SubscriptionExpiresAt     *time.Time `bun:"subscription_expires_at,nullzero"`
	IsTrial                   bool       `bun:"is_trial"`
	TrialExpiresAt            *time.Time `bun:"trial_expires_at,nullzero"`
	MaxConcurrentRounds       int        `bun:"max_concurrent_rounds"`
	MaxParticipantsPerRound   int        `bun:"max_participants_per_round"`
	CommandsPerMinute         int        `bun:"commands_per_minute"`
	RoundsPerDay              int        `bun:"rounds_per_day"`
	CustomLeaderboardsEnabled bool       `bun:"custom_leaderboards_enabled"`
}
