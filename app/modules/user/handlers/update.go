package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// UpdateUserHandler handles the UpdateUserCommand.
type UpdateUserHandler struct {
	userDB   userdb.UserDB
	eventBus *watermillutil.PubSub // Use your PubSub struct
}

// Handle processes the UpdateUserCommand.
func (h *UpdateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the UpdateUserCommand from the message payload.
	var cmd UpdateUserRequest
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

	// 3. Update the user in the database
	if err := h.userDB.UpdateUser(context.Background(), cmd.DiscordID, user); err != nil {
		return fmt.Errorf("failed to update user in database: %w", err)
	}

	// 4. Publish a UserUpdatedEvent using the publisher from PubSub
	userUpdatedEvent := UserUpdatedEvent{
		DiscordID: cmd.DiscordID,
		// ... populate other fields from cmd if needed
	}

	// Marshal the UserUpdatedEvent using encoding/json
	payload, err := json.Marshal(userUpdatedEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal UserUpdatedEvent: %w", err)
	}

	if err := h.eventBus.Publish("user.updated", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish UserUpdatedEvent: %w", err)
	}

	return nil
}
