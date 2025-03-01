package scoredb

// Score represents a score with DiscordID, RoundID, Score, and TagNumber.
type Score struct {
	DiscordID string `bun:"user_id,notnull"`
	RoundID   string `bun:"round_id,notnull"`
	Score     int    `bun:"score,notnull"`
	TagNumber int    `bun:"tag_number"`
	Source    string `bun:"source,notnull"`
}
