package leaderboardcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// TagSwapRequest represents a command to request a tag swap.
type TagSwapRequest struct {
	RequestorID  string `json:"requestor_id"`
	RequestorTag int    `json:"requestor_tag"`
	TargetTag    string `json:"target_tag"`
}

// CommandName returns the command name for TagSwapRequest.
func (cmd TagSwapRequest) CommandName() string {
	return "tag_swap_request"
}

var _ commands.Command = TagSwapRequest{}
