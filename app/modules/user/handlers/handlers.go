package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/service"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	UserService userservice.Service // Use the new interface name
	Publisher   message.Publisher
}

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) error {
	defer msg.Ack()

	var req userevents.UserSignupRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal UserSignupRequest: %w", err)
	}

	resp, err := h.UserService.OnUserSignupRequest(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to process user signup request: %w", err)
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal UserSignupResponse: %w", err)
	}

	if err := h.Publisher.Publish(userevents.UserSignupResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		return fmt.Errorf("failed to publish UserSignupResponse: %w", err)
	}

	return nil
}

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(msg *message.Message) error {
	defer msg.Ack()

	var req userevents.UserRoleUpdateRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal UserRoleUpdateRequest: %w", err)
	}

	resp, err := h.UserService.OnUserRoleUpdateRequest(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to process user role update request: %w", err)
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal UserRoleUpdateResponse: %w", err)
	}

	if err := h.Publisher.Publish(userevents.UserRoleUpdateResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdateResponse: %w", err)
	}

	return nil
}
