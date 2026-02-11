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

// Season represents a named season period for point tracking.
type Season struct {
	bun.BaseModel `bun:"table:leaderboard_seasons,alias:sn"`

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
	SeasonID  string                `bun:"season_id,notnull,default:'default'"`
	MemberID  sharedtypes.DiscordID `bun:"member_id,notnull"`
	RoundID   sharedtypes.RoundID   `bun:"round_id,type:uuid,notnull"`
	Points    int                   `bun:"points,notnull"`
	Reason    string                `bun:"reason"`     // e.g., "Round Matchups", "Admin adjustment: scorecard error"
	Tier      string                `bun:"tier"`       // Player's tier at time of calculation (Gold/Silver/Bronze)
	Opponents int                   `bun:"opponents"`  // Number of opponents beaten (base_points = opponents * 100)

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// SeasonStanding represents a snapshot of a player's standing for the season.
type SeasonStanding struct {
	bun.BaseModel `bun:"table:leaderboard_season_standings,alias:ss"`

	SeasonID      string                `bun:"season_id,pk,default:'default'"`
	MemberID      sharedtypes.DiscordID `bun:"member_id,pk"`
	TotalPoints   int                   `bun:"total_points,notnull,default:0"`
	CurrentTier   string                `bun:"current_tier,notnull,default:'Bronze'"`
	SeasonBestTag int                   `bun:"season_best_tag,notnull,default:0"`
	RoundsPlayed  int                   `bun:"rounds_played,notnull,default:0"`

	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
