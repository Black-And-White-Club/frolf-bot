package leaderboardcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// InsertTagAndDiscordIDRequest represents a command to insert a new tag and Discord ID into the leaderboard.
type InsertTagAndDiscordIDRequest struct {
	TagNumber int    `json:"tag_number"`
	DiscordID string `json:"discord_id"`
}

// CommandName returns the command name for InsertTagAndDiscordIDRequest.
func (cmd InsertTagAndDiscordIDRequest) CommandName() string {
	return "insert_user"
}

var _ commands.Command = InsertTagAndDiscordIDRequest{}
