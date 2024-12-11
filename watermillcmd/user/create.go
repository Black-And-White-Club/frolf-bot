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

// CreateUserHandler handles the CreateUserCommand.
type CreateUserHandler struct {
	userDB   userdb.UserDB
	eventBus watermillcmd.MessageBus
}

// Handle processes the CreateUserCommand.
func (h *CreateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the CreateUserCommand from the message payload.
	var cmd userapimodels.CreateUserCommand
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

	// 3. Check tag availability
	tagAvailable, err := h.checkTagAvailability(context.Background(), cmd.TagNumber)
	if err != nil {
		return errors.Wrap(err, "failed to check tag availability")
	}

	if tagAvailable {
		// 4. Create the user in the database
		user := &userdb.User{ // Use *userdb.User
			DiscordID: cmd.DiscordID,
			Name:      cmd.Name,
			Role:      userdb.UserRole(cmd.Role), // Use userdb.UserRole
			// ... other fields
		}
		if err := h.userDB.CreateUser(context.Background(), user); err != nil {
			return errors.Wrap(err, "failed to create user in database")
		}

		// 5. Publish a UserCreatedEvent using nats.PublishEvent
		userCreatedEvent := UserCreatedEvent{
			DiscordID: cmd.DiscordID,
			TagNumber: cmd.TagNumber,
			// ... add other relevant data from the command ...
		}
		if err := natsjetstream.PublishEvent(context.Background(), &userCreatedEvent, "user.created"); err != nil {
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

	// 2. Publish the request to the leaderboard module
	if err := h.eventBus.PublishEvent(ctx, "check-tag-availability", checkTagEvent); err != nil { // Use h.eventBus.Publish
		return false, errors.Wrap(err, "failed to publish check tag availability event")
	}

	// 3. Subscribe to the response topic
	responseChan := make(chan bool)
	_, err := h.eventBus.Subscribe(ctx, "tag-availability-result") // Use h.eventBus.Subscribe
	if err != nil {
		return false, errors.Wrap(err, "failed to subscribe to tag availability result")
	}

	// 4. Wait for the response
	select {
	case tagAvailable := <-responseChan:
		return tagAvailable, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
