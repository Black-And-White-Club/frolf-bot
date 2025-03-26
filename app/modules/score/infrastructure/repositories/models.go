package scoredb

import sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"

// Score represents a score with UserID, RoundID, Score, and TagNumber.
type Score struct {
	UserID    sharedtypes.DiscordID  `bun:"user_id,notnull"`
	RoundID   sharedtypes.RoundID    `bun:"round_id,notnull"`
	Score     sharedtypes.Score      `bun:"score,notnull"`
	TagNumber *sharedtypes.TagNumber `bun:"tag_number"`
	Source    string                 `bun:"source,notnull"`
}
