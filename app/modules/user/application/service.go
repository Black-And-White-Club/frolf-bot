package userservice

import (
	"log/slog"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB    userdb.UserDB
	eventBus  shared.EventBus
	logger    *slog.Logger
	eventUtil eventutil.EventUtil
}

// NewUserService creates a new UserService.
func NewUserService(db userdb.UserDB, eventBus shared.EventBus, logger *slog.Logger) (Service, error) {
	return &UserServiceImpl{
		UserDB:    db,
		eventBus:  eventBus,
		logger:    logger,
		eventUtil: eventutil.NewEventUtil(),
	}, nil
}
