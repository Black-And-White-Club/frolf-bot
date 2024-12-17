package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// UpdateUserHandler handles the UpdateUserCommand.
type UpdateUserHandler struct {
	userDB   userdb.UserDB
	eventBus watermillutil.PubSuber
}

// NewUpdateUserHandler creates a new UpdateUserHandler.
func NewUpdateUserHandler(userDB userdb.UserDB, eventBus watermillutil.PubSuber) *UpdateUserHandler {
	return &UpdateUserHandler{
		userDB:   userDB,
		eventBus: eventBus,
	}
}

// Handle processes the UpdateUserCommand.
func (h *UpdateUserHandler) Handle(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the UpdateUserCommand from the message payload.
	var cmd usercommands.UpdateUserRequest
	marshaler := watermillutil.Marshaler // Use the marshaler from userhandlers
	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal UpdateUserCommand")
	}

	// 2. Prepare the update map
	updates := make(map[string]interface{})
	for key, value := range cmd.Updates {
		switch key {
		case "Name":
			updates["name"] = value.(string)
		case "Role":
			updates["role"] = value.(string) // No need to convert to userdb.UserRole here
		// ... other cases for different fields
		default:
			return fmt.Errorf("invalid update field: %s", key)
		}
	}

	// 3. Update the user in the database
	if err := h.userDB.UpdateUser(ctx, cmd.DiscordID, updates); err != nil {
		return errors.Wrap(err, "failed to update user in database")
	}

	// 4. Publish a UserUpdatedEvent using the publisher from PubSub
	userUpdatedEvent := UserUpdatedEvent{ // Assuming you have this event struct defined
		DiscordID: cmd.DiscordID,
		// ... populate other fields from cmd if needed
	}

	payload, err := json.Marshal(userUpdatedEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal UserUpdatedEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicUpdateUser, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return errors.Wrap(err, "failed to publish UserUpdatedEvent")
	}

	return nil
}
