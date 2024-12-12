package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetUserRoleHandler handles requests for user roles.
type GetUserRoleHandler struct {
	queryService *userqueries.UserQueryService
	publisher    *watermillutil.NatsPublisher
}

// NewGetUserRoleHandler creates a new GetUserRoleHandler.
func NewGetUserRoleHandler(queryService *userqueries.UserQueryService, publisher *watermillutil.NatsPublisher) *GetUserRoleHandler {
	return &GetUserRoleHandler{
		queryService: queryService,
		publisher:    publisher,
	}
}

// Handle processes the GetUserRoleRequestEvent and publishes a UserRoleResponseEvent.
func (h *GetUserRoleHandler) Handle(msg *message.Message) error {
	// 1. Unmarshal the GetUserRoleRequestEvent
	var requestEvent GetUserRoleRequestEvent
	if err := json.Unmarshal(msg.Payload, &requestEvent); err != nil {
		return fmt.Errorf("failed to unmarshal GetUserRoleRequestEvent: %w", err)
	}

	// 2. Retrieve the user's role
	role, err := h.queryService.GetUserRole(context.Background(), requestEvent.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to get user role: %w", err)
	}

	// 3. Publish the UserRoleResponseEvent
	responseEvent := UserRoleResponseEvent{
		DiscordID: requestEvent.DiscordID,
		Role:      role,
	}
	payload, err := json.Marshal(responseEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal UserRoleResponseEvent: %w", err)
	}

	if err := h.publisher.Publish("get-user-role-response", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish UserRoleResponseEvent: %w", err)
	}

	return nil
}
