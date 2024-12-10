package usercommands

import (
	"context"
	"errors"

	"github.com/Black-And-White-Club/tcr-bot/db"
	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCommandService implements the api.CommandService interface.
type UserCommandService struct {
	userDB             db.UserDB // Use the DB interface
	natsConnectionPool *natsjetstream.NatsConnectionPool
	publisher          message.Publisher
	commandBus         watermillcmd.CommandBus
}

// CommandBus returns the command bus.
func (s *UserCommandService) CommandBus() watermillcmd.CommandBus {
	return s.commandBus
}

// NewUserCommandService creates a new UserCommandService.
func NewUserCommandService(userDB db.UserDB, natsConnectionPool *natsjetstream.NatsConnectionPool, publisher message.Publisher, commandBus watermillcmd.CommandBus) CommandService {
	return &UserCommandService{
		userDB:             userDB,
		natsConnectionPool: natsConnectionPool,
		publisher:          publisher,
		commandBus:         commandBus,
	}
}

// CreateUser handles user creation logic.
func (s *UserCommandService) CreateUser(ctx context.Context, discordID string, name string, role string, tagNumber int) error {
	// Validate input
	if discordID == "" || name == "" || role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Create and publish a CreateUserCommand to the command bus, including the tagNumber
	createUserCmd := userapimodels.CreateUserCommand{
		DiscordID: discordID,
		Role:      role,
		TagNumber: tagNumber, // Include the tagNumber in the command
	}
	return s.commandBus.Send(ctx, createUserCmd)
}

// UpdateUser updates an existing user.
func (s *UserCommandService) UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error {
	// Create and publish an UpdateUserCommand to the command bus
	updateUserCmd := userapimodels.UpdateUserCommand{
		DiscordID: discordID,
		Updates:   updates,
	}
	return s.commandBus.Send(ctx, updateUserCmd)
}
