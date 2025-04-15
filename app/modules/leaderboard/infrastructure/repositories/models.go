package leaderboarddb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ServiceUpdateSource defines where an update originated from
type ServiceUpdateSource string

// Constants for ServiceUpdateSource
const (
	ServiceUpdateSourceProcessScores ServiceUpdateSource = "process_scores"
	ServiceUpdateSourceManual        ServiceUpdateSource = "manual"
	ServiceUpdateSourceCreateUser    ServiceUpdateSource = "create_user"
	ServiceUpdateSourceAdminBatch    ServiceUpdateSource = "admin_batch"
	ServiceUpdateSourceTagSwap       ServiceUpdateSource = "tag_swap"
)

// Leaderboard represents a leaderboard with entries
type Leaderboard struct {
	bun.BaseModel       `bun:"table:leaderboards,alias:l"`
	ID                  int64                            `bun:"id,pk,autoincrement"`
	LeaderboardData     leaderboardtypes.LeaderboardData `bun:"leaderboard_data,type:jsonb,notnull"`
	IsActive            bool                             `bun:"is_active,notnull"`
	UpdateSource        ServiceUpdateSource              `bun:"update_source"`
	UpdateID            sharedtypes.RoundID              `bun:"update_id,type:uuid"`
	RequestingDiscordID sharedtypes.DiscordID            `bun:"requesting_discord_id,nullzero"`
}

// BeforeInsert is a hook that generates a UUID for UpdateID before inserting a new record
func (l *Leaderboard) BeforeInsert(ctx context.Context) error {
	if l.UpdateID == sharedtypes.RoundID(uuid.Nil) {
		l.UpdateID = sharedtypes.RoundID(uuid.New())
	}
	return nil
}

// TagAssignment represents a single tag assignment
type TagAssignment struct {
	UserID    sharedtypes.DiscordID
	TagNumber sharedtypes.TagNumber
}
