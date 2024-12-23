package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/service"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	UserService userservice.Service
	Publisher   message.Publisher
	logger      watermill.LoggerAdapter
}

func NewHandlers(userService userservice.Service, publisher message.Publisher, logger watermill.LoggerAdapter) Handlers {
	return &UserHandlers{
		UserService: userService,
		Publisher:   publisher,
		logger:      logger,
	}
}

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(ctx context.Context, msg *message.Message) error {
	msg.Ack()

	h.logger.Info("Processing UserSignupRequest", watermill.LogFields{"message_id": msg.UUID})

	var req userevents.UserSignupRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal UserSignupRequest", err, watermill.LogFields{"message_id": msg.UUID})
		return fmt.Errorf("failed to unmarshal UserSignupRequest: %w", err)
	}

	messageCtx := msg.Context() // Get the message context

	resp, err := h.UserService.OnUserSignupRequest(messageCtx, req)

	if err != nil {
		h.logger.Error("Failed to process user signup request", err, watermill.LogFields{"message_id": msg.UUID})
		return fmt.Errorf("failed to process user signup request: %w", err)
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal UserSignupResponse", err, watermill.LogFields{"message_id": msg.UUID})
		return fmt.Errorf("failed to marshal UserSignupResponse: %w", err)
	}

	if err := h.Publisher.Publish(userevents.UserSignupResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		h.logger.Error("Failed to publish UserSignupResponse", err, watermill.LogFields{"message_id": msg.UUID})
		return fmt.Errorf("failed to publish UserSignupResponse: %w", err)
	}

	h.logger.Info("HandleUserSignupRequest completed", watermill.LogFields{"message_id": msg.UUID})
	return nil
}

func (h *UserHandlers) HandleUserRoleUpdateRequest(ctx context.Context, msg *message.Message) error {
	h.logger.Info("HandleUserRoleUpdateRequest called", watermill.LogFields{"message_id": msg.UUID})

	var req userevents.UserRoleUpdateRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal UserRoleUpdateRequest", err, watermill.LogFields{"message_id": msg.UUID})
		// Consider returning a specific error type here (e.g., ErrInvalidPayload) to differentiate from service errors
		return fmt.Errorf("failed to unmarshal UserRoleUpdateRequest: %w", err)
	}

	// Use a new variable for the context derived from the message
	messageCtx := msg.Context()

	// Set a timeout context for the service call
	timeoutCtx, timeoutCancel := context.WithTimeout(messageCtx, 5*time.Second) // 5-second timeout
	defer timeoutCancel()

	resp, err := h.UserService.OnUserRoleUpdateRequest(timeoutCtx, req)
	if err != nil {
		h.logger.Error("Failed to process user role update request", err, watermill.LogFields{"message_id": msg.UUID})
		// Consider returning a specific error type here (e.g., ErrUserService) to differentiate from other errors
		// You can check for specific error types returned by the service and handle them accordingly
		return fmt.Errorf("failed to process user role update request: %w", err)
	}

	// Publish the response event
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal UserRoleUpdateResponse", err, watermill.LogFields{"message_id": msg.UUID})
		// Consider returning a specific error type here (e.g., ErrMarshalling) to differentiate from other errors
		return fmt.Errorf("failed to marshal UserRoleUpdateResponse: %w", err)
	}

	if err := h.Publisher.Publish(userevents.UserRoleUpdateResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		h.logger.Error("Failed to publish UserRoleUpdateResponse", err, watermill.LogFields{"message_id": msg.UUID})
		// Consider returning a specific error type here (e.g., ErrPublish) to differentiate from other errors
		return fmt.Errorf("failed to publish UserRoleUpdateResponse: %w", err)
	}

	h.logger.Info("HandleUserRoleUpdateRequest completed", watermill.LogFields{"message_id": msg.UUID})
	return nil
}
