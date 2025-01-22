package userhandlers

import (
	"log/slog"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	userService userservice.Service
	logger      *slog.Logger
}

// NewUserHandlers creates a new UserHandlers.
func NewUserHandlers(userService userservice.Service, logger *slog.Logger) *UserHandlers {
	return &UserHandlers{
		userService: userService,
		logger:      logger,
	}
}
