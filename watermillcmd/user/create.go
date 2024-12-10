package userhandlers

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/db"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors" // Import the errors package
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

)

type CreateUserHandler struct {
	userDB   db.UserDB
	eventBus watermillcmd.EventBus
}

func (h *CreateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the CreateUserCommand from the message payload.
	var cmd userapimodels.CreateUserCommand // Use the correct command struct
  if err := cqrs.ProtobufMarshaler{}.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateUserCommand") // Wrap the error
	}

	// Check if a user with this Discord ID already exists
	existingUser, err := h.userDB.GetUserByDiscordID(context.Background(), cmd.DiscordID)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing user") // Wrap the error
	}

	if existingUser != nil {
		// User already exists, handle accordingly (e.g., return an error or update the user)
		return fmt.Errorf("user with Discord ID %s already exists", cmd.DiscordID)
	}

	// Check tag availability (replace with your actual logic)
	tagAvailable, err := h.checkTagAvailability(cmd.TagNumber)
	if err != nil {
		return errors.Wrap(err, "failed to check tag availability") // Wrap the error
	}

	if tagAvailable {
		// Create the user in the database
		err := h.userDB.CreateUser(context.Background(), &db.User{
			DiscordID: cmd.DiscordID,
			Name:      cmd.Name,
			Role:      db.UserRole(cmd.Role), // Convert to db.UserRole
			TagNumber: cmd.TagNumber,
			// ... other fields
		})
		if err != nil {
			return errors.Wrap(err, "failed to create user in database") // Wrap the error
		}

		// Publish a UserCreatedEvent using nats.PublishEvent
		if err := nats.PublishEvent(context.Background(), &UserCreatedEvent{
			DiscordID: cmd.DiscordID,
			Name:      cmd.Name,
			Role:      db.UserRole(cmd.Role), // Convert to db.UserRole
			TagNumber: cmd.TagNumber,
			// ... add other relevant data from the command ...
		}); err != nil {
			return errors.Wrap(err, "failed to publish UserCreatedEvent") // Wrap the error
		}
	} else {
		return fmt.Errorf("tag number %d is already in use", cmd.TagNumber)
	}

	return nil
}

// checkTagAvailability is a placeholder for your tag availability logic.
func (h *CreateUserHandler) checkTagAvailability(tagNumber int) (bool, error) {
	// Replace this with your actual implementation
	// This example always returns true for simplicity
	return true, nil
}
