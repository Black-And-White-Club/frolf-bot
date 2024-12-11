package userhandlers

import (
	"context"
	"fmt"

	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// UpdateUserHandler handles the UpdateUserCommand.
type UpdateUserHandler struct {
	userDB   userdb.UserDB
	eventBus watermillcmd.MessageBus
}

// Handle processes the UpdateUserCommand.
func (h *UpdateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the UpdateUserCommand from the message payload.
	var cmd userapimodels.UpdateUserCommand
	marshaler := cqrs.JSONMarshaler{}

	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal UpdateUserCommand")
	}

	// 2. Create a userdb.User from the data in cmd.Updates
	user := &userdb.User{
		DiscordID: cmd.DiscordID,
	}
	for key, value := range cmd.Updates {
		switch key {
		case "Name":
			user.Name = value.(string)
		case "Role":
			user.Role = userdb.UserRole(value.(string)) // Convert to userdb.UserRole
			// ... other cases for different fields
		}
	}

	// 3. Update the user in the database using the userDB.
	if err := h.userDB.UpdateUser(context.Background(), cmd.DiscordID, user); err != nil { // Pass cmd.DiscordID
		return fmt.Errorf("failed to update user in database: %w", err)
	}

	// 4. Publish a UserUpdatedEvent using natsjetstream.PublishEvent.
	userUpdatedEvent := UserUpdatedEvent{
		DiscordID: cmd.DiscordID,
		// ... populate other fields from cmd if needed
	}
	if err := natsjetstream.PublishEvent(context.Background(), &userUpdatedEvent, "user.updated"); err != nil {
		return fmt.Errorf("failed to publish UserUpdatedEvent: %w", err)
	}

	return nil
}
