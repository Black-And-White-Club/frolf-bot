package leaderboarddb

import (
	"time"

	"github.com/uptrace/bun"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// Season represents a named season period for point tracking.
type Season struct {
	bun.BaseModel `bun:"table:leaderboard_seasons,alias:sn"`

	GuildID   string    `bun:"guild_id,pk,notnull,default:''"`
	ID        string    `bun:"id,pk"`        // e.g., "2026-spring"
	Name      string    `bun:"name,notnull"` // Display name e.g., "Spring 2026"
	IsActive  bool      `bun:"is_active,notnull,default:false"`
	StartDate time.Time `bun:"start_date,nullzero"`
	EndDate   time.Time `bun:"end_date,nullzero"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// PointHistory records points earned by a member for a specific round.
type PointHistory struct {
	bun.BaseModel `bun:"table:leaderboard_point_history,alias:ph"`

	ID        int64                 `bun:"id,pk,autoincrement"`
	GuildID   string                `bun:"guild_id,notnull,default:''"`
	SeasonID  string                `bun:"season_id,notnull,default:'default'"`
	MemberID  sharedtypes.DiscordID `bun:"member_id,notnull"`
	RoundID   sharedtypes.RoundID   `bun:"round_id,type:uuid,notnull"`
	Points    int                   `bun:"points,notnull"`
	Reason    string                `bun:"reason"`    // e.g., "Round Matchups", "Admin adjustment: scorecard error"
	Tier      string                `bun:"tier"`      // Player's tier at time of calculation (Gold/Silver/Bronze)
	Opponents int                   `bun:"opponents"` // Number of opponents beaten (base_points = opponents * 100)

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// SeasonStanding represents a snapshot of a player's standing for the season.
type SeasonStanding struct {
	bun.BaseModel `bun:"table:leaderboard_season_standings,alias:ss"`

	GuildID       string                `bun:"guild_id,pk,notnull,default:''"`
	SeasonID      string                `bun:"season_id,pk,default:'default'"`
	MemberID      sharedtypes.DiscordID `bun:"member_id,pk"`
	TotalPoints   int                   `bun:"total_points,notnull,default:0"`
	CurrentTier   string                `bun:"current_tier,notnull,default:'Bronze'"`
	SeasonBestTag int                   `bun:"season_best_tag,notnull,default:0"`
	RoundsPlayed  int                   `bun:"rounds_played,notnull,default:0"`

	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// LeagueMember represents persistent tag state for a guild member.
type LeagueMember struct {
	bun.BaseModel `bun:"table:league_members,alias:lm"`

	GuildID      string    `bun:"guild_id,pk,notnull"`
	MemberID     string    `bun:"member_id,pk,notnull"`
	CurrentTag   *int      `bun:"current_tag"` // NULL = no tag
	LastActiveAt time.Time `bun:"last_active_at,notnull,default:now()"`
	UpdatedAt    time.Time `bun:"updated_at,notnull,default:now()"`
}

// TagHistoryEntry represents an immutable record in the tag ledger.
type TagHistoryEntry struct {
	bun.BaseModel `bun:"table:tag_history,alias:th"`

	ID          int64      `bun:"id,pk,autoincrement"`
	GuildID     string     `bun:"guild_id,notnull"`
	RoundID     *uuid.UUID `bun:"round_id,type:uuid"`
	TagNumber   int        `bun:"tag_number,notnull"`
	OldMemberID *string    `bun:"old_member_id"`
	NewMemberID string     `bun:"new_member_id,notnull"`
	Reason      string     `bun:"reason,notnull"` // claim|round_swap|admin_fix|reset
	Metadata    string     `bun:"metadata,type:jsonb,notnull,default:'{}'"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:now()"`
}

// RoundOutcome tracks idempotency and recalculation for processed rounds.
type RoundOutcome struct {
	bun.BaseModel `bun:"table:leaderboard_round_outcomes,alias:ro"`

	GuildID        string    `bun:"guild_id,pk,notnull"`
	RoundID        uuid.UUID `bun:"round_id,pk,type:uuid,notnull"`
	SeasonID       *string   `bun:"season_id"`
	ProcessingHash string    `bun:"processing_hash,notnull"`
	ProcessedAt    time.Time `bun:"processed_at,notnull,default:now()"`
}
