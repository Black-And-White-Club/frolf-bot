package usercommands

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// UserService defines the interface for user commands.
type CommandService interface {
	CreateUser(ctx context.Context, discordID string, name string, role string, tagNumber int) error
	UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error
	// Add other command methods as needed (e.g., DeleteUser)

	CommandBus() cqrs.CommandBus // Use cqrs.CommandBus directly
}
