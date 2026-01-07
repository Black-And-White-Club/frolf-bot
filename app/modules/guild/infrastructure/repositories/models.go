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
	// Snapshot of resource IDs and deletion results. Stored as JSONB so the Discord
	// worker can safely perform deletions after the main transaction commits.
	ResourceState      *ResourceState `bun:"resource_state,type:jsonb" json:"resource_state,omitempty"`
	SignupEmoji        string         `bun:"signup_emoji,notnull,default:'🐍',type:varchar(10)"`
	AutoSetupCompleted bool           `bun:"auto_setup_completed,notnull,default:false"`
	SetupCompletedAt   *time.Time     `bun:"setup_completed_at,nullzero"`
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

// ResourceState represents a snapshot of bot-created Discord resources for the
// guild and per-resource deletion results. This is stored in `guild_configs.resource_state`
// as JSONB and is intended to be written before explicit resource ID columns are
// nullified during a reset operation.
type ResourceState struct {
	SignupChannelID      string `json:"signup_channel_id,omitempty"`
	SignupMessageID      string `json:"signup_message_id,omitempty"`
	EventChannelID       string `json:"event_channel_id,omitempty"`
	LeaderboardChannelID string `json:"leaderboard_channel_id,omitempty"`
	UserRoleID           string `json:"user_role_id,omitempty"`
	EditorRoleID         string `json:"editor_role_id,omitempty"`
	AdminRoleID          string `json:"admin_role_id,omitempty"`
	// Results keyed by resource name (e.g., "signup_channel") describing deletion outcomes.
	Results map[string]DeletionResult `json:"results,omitempty"`
}

// IsEmpty reports whether the ResourceState contains any meaningful data.
// It is safe to call on a nil receiver.
func (rs *ResourceState) IsEmpty() bool {
	if rs == nil {
		return true
	}
	if rs.SignupChannelID != "" {
		return false
	}
	if rs.SignupMessageID != "" {
		return false
	}
	if rs.EventChannelID != "" {
		return false
	}
	if rs.LeaderboardChannelID != "" {
		return false
	}
	if rs.UserRoleID != "" {
		return false
	}
	if rs.EditorRoleID != "" {
		return false
	}
	if rs.AdminRoleID != "" {
		return false
	}
	if len(rs.Results) != 0 {
		return false
	}
	return true
}

// DeletionResult records the outcome of an attempted deletion for a single resource.
type DeletionResult struct {
	Status    string     `json:"status"`          // e.g., "pending", "success", "failed"
	Error     string     `json:"error,omitempty"` // error message if any
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
