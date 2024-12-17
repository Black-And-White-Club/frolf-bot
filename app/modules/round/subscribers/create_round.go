// In app/modules/round/commands/edit_round.go

package roundcommands

import (
	"errors"

	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// CreateRoundRequest represents the request to create a new round.
type CreateRoundRequest struct {
	Input rounddto.CreateRoundInput `json:"input"`
}

// Validate checks if the command is valid.
func (cmd CreateRoundRequest) Validate() error {
	if cmd.Input.Title == "" {
		return errors.New("title is required")
	}
	// ... other validations you might want to add ...
	return nil
}

// CommandName returns the name of the command.
func (cmd CreateRoundRequest) CommandName() string {
	return "create_round"
}

var _ commands.Command = CreateRoundRequest{}
