package guilddb

import (
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// GuildConfig is the Bun ORM model for the guild_configs table.
type GuildConfig struct {
	bun.BaseModel `bun:"table:guild_configs,alias:g"`

	GuildID   sharedtypes.GuildID `bun:"guild_id,pk,notnull,type:varchar(20)"`
	CreatedAt time.Time           `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time           `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	IsActive  bool                `bun:"is_active,notnull,default:true"`

	// Tracks deletion lifecycle. Plain string with default to avoid pointer noise.
	DeletionStatus string `bun:"deletion_status,type:deletion_status_enum,notnull,default:'none'"`

	// Discord resource IDs
	SignupChannelID      string `bun:"signup_channel_id,nullzero,type:varchar(20)"`
	SignupMessageID      string `bun:"signup_message_id,nullzero,type:varchar(20)"`
	EventChannelID       string `bun:"event_channel_id,nullzero,type:varchar(20)"`
	LeaderboardChannelID string `bun:"leaderboard_channel_id,nullzero,type:varchar(20)"`
	UserRoleID           string `bun:"user_role_id,nullzero,type:varchar(20)"`
	EditorRoleID         string `bun:"editor_role_id,nullzero,type:varchar(20)"`
	AdminRoleID          string `bun:"admin_role_id,nullzero,type:varchar(20)"`

	// Snapshot of resource IDs stored as JSONB for background cleanup.
	ResourceState *ResourceState `bun:"resource_state,type:jsonb" json:"resource_state,omitempty"`

	// varchar(64) to support custom Discord emojis like <:frolf:123456789012345678>
	SignupEmoji        string     `bun:"signup_emoji,notnull,default:'🐍',type:varchar(64)"`
	AutoSetupCompleted bool       `bun:"auto_setup_completed,notnull,default:false"`
	SetupCompletedAt   *time.Time `bun:"setup_completed_at,nullzero"`

	// Subscription and rate limiting
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

// ResourceState represents a snapshot of bot-created Discord resources.
type ResourceState struct {
	SignupChannelID      string                    `json:"signup_channel_id,omitempty"`
	SignupMessageID      string                    `json:"signup_message_id,omitempty"`
	EventChannelID       string                    `json:"event_channel_id,omitempty"`
	LeaderboardChannelID string                    `json:"leaderboard_channel_id,omitempty"`
	UserRoleID           string                    `json:"user_role_id,omitempty"`
	EditorRoleID         string                    `json:"editor_role_id,omitempty"`
	AdminRoleID          string                    `json:"admin_role_id,omitempty"`
	Results              map[string]DeletionResult `json:"results,omitempty"`
}

// DeletionResult records the outcome of an attempted deletion for a single resource.
type DeletionResult struct {
	Status    string     `json:"status"`
	Error     string     `json:"error,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
