package usercommands

import (
	"context"
	"errors"
	"log"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCommandService implements the api.CommandService interface.
type UserCommandService struct {
	UserDB     userdb.UserDB
	publisher  message.Publisher
	commandBus cqrs.CommandBus // Use cqrs.CommandBus directly
}

// CommandBus returns the command bus.
func (s *UserCommandService) CommandBus() cqrs.CommandBus { // Use cqrs.CommandBus directly
	return s.commandBus
}

// NewUserCommandService creates a new UserCommandService.
func NewUserCommandService(userDB userdb.UserDB, publisher message.Publisher, commandBus cqrs.CommandBus) CommandService { // Use cqrs.CommandBus directly
	return &UserCommandService{
		UserDB:     userDB,
		publisher:  publisher,
		commandBus: commandBus,
	}
}

// CreateUser handles user creation logic.
func (s *UserCommandService) CreateUser(ctx context.Context, discordID string, name string, role string, tagNumber int) error {
	// Validate input
	if discordID == "" || name == "" || role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Create and publish a CreateUserCommand to the command bus, including the tagNumber
	createUserCmd := userhandlers.CreateUserRequest{
		DiscordID: discordID,
		Name:      name,
		Role:      role,
		TagNumber: tagNumber,
	}

	// Log the command being sent
	log.Printf("Sending CreateUserCommand: %+v\n", createUserCmd)

	return s.commandBus.Send(ctx, createUserCmd)
}

// UpdateUser updates an existing user.
func (s *UserCommandService) UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error {
	// Create and publish an UpdateUserCommand to the command bus
	updateUserCmd := userhandlers.UpdateUserRequest{
		DiscordID: discordID,
		Updates:   updates,
	}

	// Log the command being sent
	log.Printf("Sending UpdateUserCommand: %+v\n", updateUserCmd)

	return s.commandBus.Send(ctx, updateUserCmd)
}
