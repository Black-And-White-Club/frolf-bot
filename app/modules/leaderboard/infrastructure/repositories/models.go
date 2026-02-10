package leaderboarddb

import (
	"context"
	"time"

	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// Leaderboard represents a leaderboard with entries
type Leaderboard struct {
	bun.BaseModel   `bun:"table:leaderboards,alias:l"`
	ID              int64                            `bun:"id,pk,autoincrement"`
	LeaderboardData leaderboardtypes.LeaderboardData `bun:"leaderboard_data,type:jsonb,notnull"`
	IsActive        bool                             `bun:"is_active,notnull"`
	UpdateSource    sharedtypes.ServiceUpdateSource  `bun:"update_source"`
	UpdateID        sharedtypes.RoundID              `bun:"update_id,type:uuid"`
	GuildID         sharedtypes.GuildID              `bun:"guild_id,notnull"`
}

var _ bun.BeforeInsertHook = (*Leaderboard)(nil)

func (l *Leaderboard) BeforeInsert(ctx context.Context, _ *bun.InsertQuery) error {
	if uuid.UUID(l.UpdateID) == uuid.Nil {
		l.UpdateID = sharedtypes.RoundID(uuid.New())
	}
	return nil
}

// PointHistory records points earned by a member for a specific round.
type PointHistory struct {
	bun.BaseModel `bun:"table:leaderboard_point_history,alias:ph"`

	ID       int64                 `bun:"id,pk,autoincrement"`
	MemberID sharedtypes.DiscordID `bun:"member_id,notnull"`
	RoundID  sharedtypes.RoundID   `bun:"round_id,type:uuid,notnull"`
	Points   int                   `bun:"points,notnull"`
	Reason   string                `bun:"reason"` // e.g., "Win vs Player X", "Giant Slayer Bonus"

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// SeasonStanding represents a snapshot of a player's standing for the season.
// This can be a materialized view or a standard table updated per round.
type SeasonStanding struct {
	bun.BaseModel `bun:"table:leaderboard_season_standings,alias:ss"`

	MemberID      sharedtypes.DiscordID `bun:"member_id,pk"`
	TotalPoints   int                   `bun:"total_points,notnull,default:0"`
	CurrentTier   string                `bun:"current_tier,notnull,default:'Bronze'"`
	SeasonBestTag int                   `bun:"season_best_tag,notnull,default:0"`
	RoundsPlayed  int                   `bun:"rounds_played,notnull,default:0"`

	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
