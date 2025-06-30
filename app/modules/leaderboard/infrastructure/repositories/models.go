package leaderboarddb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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

// BeforeInsert is a hook that generates a UUID for UpdateID before inserting a new record
func (l *Leaderboard) BeforeInsert(ctx context.Context) error {
	if l.UpdateID == sharedtypes.RoundID(uuid.Nil) {
		l.UpdateID = sharedtypes.RoundID(uuid.New())
	}
	return nil
}

// TagAssignment represents a single tag assignment for database operations
type TagAssignment struct {
	UserID    sharedtypes.DiscordID
	TagNumber sharedtypes.TagNumber
}

// FindEntryForUser returns a pointer to the entry for the given user, or nil if not found.
func (l *Leaderboard) FindEntryForUser(userID sharedtypes.DiscordID) *leaderboardtypes.LeaderboardEntry {
	for i := range l.LeaderboardData {
		if l.LeaderboardData[i].UserID == userID {
			return &l.LeaderboardData[i]
		}
	}
	return nil
}
