package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	UserService userservice.Service
	EventBus    shared.EventBus
	logger      shared.LoggerAdapter
}

// NewHandlers creates a new UserHandlers.
func NewHandlers(userService userservice.Service, eventBus shared.EventBus, logger shared.LoggerAdapter) user.Handlers {
	return &UserHandlers{
		UserService: userService,
		EventBus:    eventBus,
		logger:      logger,
	}
}

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(ctx context.Context, msg shared.Message) error {
	h.logger.Info("HandleUserSignupRequest started", shared.LogFields{"contextErr": ctx.Err()})

	h.logger.Info("Processing UserSignupRequest", shared.LogFields{"message_id": msg.UUID()})

	var req userevents.UserSignupRequestPayload
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		h.logger.Error("Failed to unmarshal UserSignupRequest", err, shared.LogFields{
			"message_id": msg.UUID(),
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to unmarshal UserSignupRequest: %w", err)
	}

	// Call the service to handle user signup
	resp, err := h.UserService.OnUserSignupRequest(msg.Context(), req)
	if err != nil {
		h.logger.Error("Failed to process user signup request", err, shared.LogFields{
			"message_id": msg.UUID(),
			"error":      err.Error(),
			"discord_id": req.DiscordID,
			"tag_number": req.TagNumber,
		})

		// Publish a UserSignupResponse with the error
		errorResponse := &userevents.UserSignupResponsePayload{
			Success: false,
			Error:   err.Error(),
		}
		if err := h.publishEvent(ctx, userevents.UserSignupResponse, errorResponse); err != nil {
			return fmt.Errorf("failed to publish UserSignupResponse event: %w", err)
		}
	} else {
		// Publish a successful UserSignupResponse
		if err := h.publishEvent(ctx, userevents.UserSignupResponse, resp); err != nil {
			return fmt.Errorf("failed to publish UserSignupResponse event: %w", err)
		}
	}

	h.logger.Info("HandleUserSignupRequest completed", shared.LogFields{"message_id": msg.UUID()})
	msg.Ack()
	return nil
}

func (h *UserHandlers) HandleUserRoleUpdateRequest(ctx context.Context, msg shared.Message) error {
	h.logger.Info("HandleUserRoleUpdateRequest started", shared.LogFields{
		"message_id": msg.UUID(),
		"contextErr": ctx.Err(),
	})

	var req userevents.UserRoleUpdateRequestPayload
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		h.logger.Error("Failed to unmarshal UserRoleUpdateRequest", err, shared.LogFields{
			"message_id": msg.UUID(),
		})
		return fmt.Errorf("failed to unmarshal UserRoleUpdateRequest: %w", err)
	}

	// Validate UserRole field
	if !req.NewRole.IsValid() {
		err := fmt.Errorf("invalid UserRole: %s", req.NewRole.String())
		h.logger.Error("Validation failed for UserRoleUpdateRequest", err, shared.LogFields{
			"message_id": msg.UUID(),
			"new_role":   req.NewRole.String(),
		})
		return fmt.Errorf("validation error: %w", err)
	}

	// Set a timeout context for the service call
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()

	resp, err := h.UserService.OnUserRoleUpdateRequest(timeoutCtx, req)
	if err != nil {
		// Log the specific error and a general message
		h.logger.Error("Failed to process user role update request", err, shared.LogFields{
			"message_id": msg.UUID(),
			"error_msg":  err.Error(), // Add the error message to the log fields
		})
		return fmt.Errorf("failed to process user role update request: %w", err)
	}

	// Publish the response event using the EventBus
	if err := h.publishEvent(ctx, userevents.UserRoleUpdateResponse, resp); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdateResponse event: %w", err)
	}

	h.logger.Info("HandleUserRoleUpdateRequest completed successfully", shared.LogFields{"message_id": msg.UUID()})
	msg.Ack()
	return nil
}

// Helper function to publish events (updated to use shared.Message)
func (h *UserHandlers) publishEvent(ctx context.Context, eventType shared.EventType, payload interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadData)
	return h.EventBus.Publish(ctx, eventType, msg)
}
