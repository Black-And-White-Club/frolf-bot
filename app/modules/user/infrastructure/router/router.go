package userrouter

import (
	"context"
	"errors"
	"log/slog"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
)

type UserRouter struct {
	router     *message.Router
	logger     *slog.Logger
	subscriber message.Subscriber
}

func NewUserRouter(logger *slog.Logger) *UserRouter {
	return &UserRouter{
		logger: logger,
	}
}

func (r *UserRouter) Configure(
	router *message.Router, // Receive the main router
	userService userservice.Service,
) error {
	// Create UserHandlers, passing in the userService
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger)

	// Register handlers for user events
	router.AddNoPublisherHandler(
		"handle-user-signup-request",
		userevents.UserSignupRequest,
		r.subscriber,
		userHandlers.HandleUserSignupRequest,
	)
	router.AddNoPublisherHandler(
		"handle-user-permissions-check-response",
		userevents.UserPermissionsCheckResponse,
		r.subscriber,
		userHandlers.HandleUserPermissionsCheckResponse,
	)
	router.AddNoPublisherHandler(
		"handle-user-role-update-request",
		userevents.UserRoleUpdateRequest,
		r.subscriber,
		userHandlers.HandleUserRoleUpdateRequest,
	)
	router.AddNoPublisherHandler(
		"handle-user-creation-failed",
		userevents.UserCreationFailed,
		r.subscriber,
		userHandlers.HandleUserCreationFailed,
	)
	router.AddNoPublisherHandler(
		"handle-user-role-update-failed",
		userevents.UserRoleUpdateFailed,
		r.subscriber,
		userHandlers.HandleUserRoleUpdateFailed,
	)
	router.AddNoPublisherHandler(
		"handle-get-user-request",
		userevents.GetUserRequest,
		r.subscriber,
		userHandlers.HandleGetUserRequest,
	)
	router.AddNoPublisherHandler(
		"handle-user-permissions-check-request",
		userevents.UserPermissionsCheckRequest,
		r.subscriber,
		userHandlers.HandleUserPermissionsCheckRequest,
	)
	router.AddNoPublisherHandler(
		"handle-user-permissions-check-failed",
		userevents.UserPermissionsCheckFailed,
		r.subscriber,
		userHandlers.HandleUserPermissionsCheckFailed,
	)
	router.AddNoPublisherHandler(
		"handle-get-user-role-request",
		userevents.GetUserRoleRequest,
		r.subscriber,
		userHandlers.HandleGetUserRoleRequest,
	)
	router.AddNoPublisherHandler(
		"handle-check-tag-availability-request",
		userevents.LeaderboardTagAvailabilityCheckRequest,
		r.subscriber,
		userHandlers.HandleCheckTagAvailabilityRequest,
	)

	r.router = router
	return nil
}

func (r *UserRouter) Run(ctx context.Context) error {
	if r.router == nil {
		return errors.New("router is not configured")
	}
	return r.router.Run(ctx)
}

func (r *UserRouter) Close() error {
	if r.router == nil {
		return errors.New("router is not initialized")
	}
	return r.router.Close()
}
