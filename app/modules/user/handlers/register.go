package userhandlers

import (
	"log"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// RegisterUserCommandHandlers registers all the command handlers for the user module.
func RegisterUserCommandHandlers(
	commandBus *cqrs.CommandBus,
	userDB userdb.UserDB,
	eventBus *watermillutil.PubSub,
) error {
	// Log an informative message indicating that the handlers are being registered
	log.Println("Registering user command handlers...")

	// No need to call AddHandlers or RegisterHandler here

	// The handlers are already added when creating the CommandBus with NewCommandBusWithConfig

	return nil
}
