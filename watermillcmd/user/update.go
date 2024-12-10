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
)

type UpdateUserHandler struct {
	userDB   userdb.UserDB
	eventBus watermillcmd.EventBus
}

func (h *UpdateUserHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the UpdateUserCommand from the message payload.
	var cmd userapimodels.UpdateUserCommand
  if err := cqrs.ProtobufMarshaler{}.Unmarshal(msg, &cmd); err != nil {
    return errors.Wrap(err, "failed to unmarshal UpdateUserCommand") // Wrap the error
  }

	// 2. Update the user in the database using the userDB.
	err := h.userDB.UpdateUser(context.Background(), cmd.DiscordID, cmd.Updates)
	if err != nil {
		return fmt.Errorf("failed to update user in database: %w", err)
	}

	// 3. Publish a UserUpdatedEvent using nats.PublishEvent
	if err := natsjetstream.PublishEvent(context.Background(), &UserUpdatedEvent{
		DiscordID: cmd.DiscordID,
		Updates:   cmd.Updates,
		// ... add other relevant data from the command ...
	}); err != nil {
		return fmt.Errorf("failed to publish UserUpdatedEvent: %w", err)
	}

	return nil
}
