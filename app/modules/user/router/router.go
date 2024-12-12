package userrouter

import (
	"context"
	"fmt"
	"log"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCommandBus is the command bus for the user module.
type UserCommandBus struct {
	publisher message.Publisher
	marshaler cqrs.CommandEventMarshaler
}

// NewUserCommandBus creates a new UserCommandBus.
func NewUserCommandBus(publisher message.Publisher, marshaler cqrs.CommandEventMarshaler) *UserCommandBus {
	return &UserCommandBus{publisher: publisher, marshaler: marshaler}
}

func (r UserCommandBus) Send(ctx context.Context, cmd usercommands.Command) error {
	msg, err := r.marshaler.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	return r.publisher.Publish(cmd.CommandName(), msg)
}

// UserCommandRouter implements the CommandRouter interface.
type UserCommandRouter struct {
	commandBus CommandBus
}

// NewUserCommandRouter creates a new UserCommandRouter.
func NewUserCommandRouter(commandBus CommandBus) CommandRouter {
	return &UserCommandRouter{commandBus: commandBus}
}

// CreateUser handles user creation logic.
func (s *UserCommandRouter) CreateUser(ctx context.Context, discordID string, name string, role userdb.UserRole, tagNumber int) error {
	createUserCmd := usercommands.CreateUserRequest{
		DiscordID: discordID,
		Name:      name,
		Role:      role,
		TagNumber: tagNumber,
	}

	log.Printf("Sending CreateUserCommand: %+v\n", createUserCmd)

	return s.commandBus.Send(ctx, createUserCmd)
}

// UpdateUser updates an existing user.
func (s *UserCommandRouter) UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error {
	updateUserCmd := usercommands.UpdateUserRequest{
		DiscordID: discordID,
		Updates:   updates,
	}

	log.Printf("Sending UpdateUserCommand: %+v\n", updateUserCmd)

	return s.commandBus.Send(ctx, updateUserCmd)
}
