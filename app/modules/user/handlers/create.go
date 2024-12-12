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

// CreateUserHandler handles the CreateUserCommand.
type CreateUserHandler struct {
	userDB   userdb.UserDB
	eventBus *watermillutil.PubSub // Use your PubSub struct
}

// Handle processes the CreateUserCommand.
func (h *CreateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the CreateUserCommand from the message payload.
	var cmd CreateUserRequest
	marshaler := cqrs.JSONMarshaler{}

	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateUserCommand")
	}

	// 2. Check if a user with this Discord ID already exists
	existingUser, err := h.userDB.GetUserByDiscordID(context.Background(), cmd.DiscordID)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing user")
	}

	if existingUser != nil {
		return fmt.Errorf("user with Discord ID %s already exists", cmd.DiscordID)
	}

	// 3. Check tag availability using PubSub
	tagAvailable, err := h.checkTagAvailability(context.Background(), cmd.TagNumber)
	if err != nil {
		return errors.Wrap(err, "failed to check tag availability")
	}

	if tagAvailable {
		// 4. Create the user in the database
		user := &userdb.User{
			DiscordID: cmd.DiscordID,
			Name:      cmd.Name,
			Role:      userdb.UserRole(cmd.Role),
			// ... other fields
		}
		if err := h.userDB.CreateUser(context.Background(), user); err != nil {
			return errors.Wrap(err, "failed to create user in database")
		}

		// 5. Publish a UserCreatedEvent using the publisher from PubSub
		userCreatedEvent := UserCreatedEvent{
			DiscordID: cmd.DiscordID,
			TagNumber: cmd.TagNumber,
			// ... add other relevant data from the command ...
		}

		// Marshal the UserCreatedEvent using encoding/json
		payload, err := json.Marshal(userCreatedEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal UserCreatedEvent: %w", err)
		}

		if err := h.eventBus.Publish("user.created", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
			return errors.Wrap(err, "failed to publish UserCreatedEvent")
		}
	} else {
		return fmt.Errorf("tag number %d is already in use", cmd.TagNumber)
	}

	return nil
}

// checkTagAvailability checks if the tag number is available.
func (h *CreateUserHandler) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// 1. Prepare the request data for the leaderboard module
	checkTagEvent := &CheckTagAvailabilityEvent{
		TagNumber: tagNumber,
	}

	// 2. Publish the request using PubSub
	payload, err := json.Marshal(checkTagEvent) // Marshal using encoding/json
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityEvent: %w", err)
	}

	if err := h.eventBus.Publish("check-tag-availability", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return false, errors.Wrap(err, "failed to publish check tag availability event")
	}

	// 3. Subscribe to the response topic using PubSub
	responseChan, err := h.eventBus.Subscribe(context.Background(), "tag-availability-result")
	if err != nil {
		return false, errors.Wrap(err, "failed to subscribe to tag availability result")
	}

	// 4. Wait for the response
	select {
	case msg := <-responseChan:
		// Unmarshal the response from msg.Payload
		var tagAvailable bool
		if err := json.Unmarshal(msg.Payload, &tagAvailable); err != nil {
			return false, fmt.Errorf("failed to unmarshal tag availability response: %w", err)
		}
		return tagAvailable, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
