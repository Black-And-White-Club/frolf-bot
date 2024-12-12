package modules

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"

	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// ModuleRegistry stores and manages application modules.
type ModuleRegistry struct {
	UserModule *user.UserModule
	// ... other modules ...
}

// NewModuleRegistry initializes and returns a new ModuleRegistry.
func NewModuleRegistry(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*ModuleRegistry, error) {
	userModule, err := user.NewUserModule(dbService, commandBus, pubsub) // Call user.NewUserModule
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user module: %w", err)
	}
	// ... initialize other modules ...

	return &ModuleRegistry{
		UserModule: userModule,
		// ... other modules ...
	}, nil
}
