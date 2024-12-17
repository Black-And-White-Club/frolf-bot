// app/modules/leaderboard/commands/manual_tag_assignment.go
package leaderboardcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// ManualTagAssignmentRequest represents a command to manually assign a tag.
type ManualTagAssignmentRequest struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// CommandName returns the command name for ManualTagAssignmentRequest
func (cmd ManualTagAssignmentRequest) CommandName() string {
	return "manual_tag_assignment"
}

var _ commands.Command = ManualTagAssignmentRequest{}
