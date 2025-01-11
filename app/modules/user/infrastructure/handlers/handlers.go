package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	UserService userservice.Service
	EventBus    shared.EventBus
	logger      *slog.Logger
}

// NewHandlers creates a new UserHandlers.
func NewHandlers(userService userservice.Service, eventBus shared.EventBus, logger *slog.Logger) user.Handlers {
	return &UserHandlers{
		UserService: userService,
		EventBus:    eventBus,
		logger:      logger,
	}
}

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(ctx context.Context, msg *message.Message) error {
	h.logger.Info("HandleUserSignupRequest started", "contextErr", ctx.Err())
	h.logger.Info("Processing UserSignupRequest", "payload", string(msg.Payload))
	h.logger.Info("Processing UserSignupRequest", "message_id", msg.UUID)

	var req userevents.UserSignupRequestPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal UserSignupRequest", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal UserSignupRequest: %w", err)
	}

	// Call the service to handle user signup
	resp, err := h.UserService.OnUserSignupRequest(msg.Context(), req)
	if err != nil {
		h.logger.Error("Failed to process user signup request", "error", err, "message_id", msg.UUID, "discord_id", req.DiscordID, "tag_number", req.TagNumber)

		// Publish a UserSignupResponse with the error
		errorResponse := &userevents.UserSignupResponsePayload{
			Success: false,
			Error:   err.Error(),
		}
		if err := h.publishEvent(ctx, userevents.UserSignupResponse, errorResponse); err != nil {
			return fmt.Errorf("failed to publish UserSignupResponse event: %w", err)
		}
	} else {
		// Publish a successful UserSignupResponse directly
		payloadData, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("failed to marshal UserSignupResponse payload: %w", err)
		}

		responseMsg := message.NewMessage(watermill.NewUUID(), payloadData)
		responseMsg.Metadata.Set("subject", userevents.UserSignupResponse)

		if err := h.EventBus.Publish(ctx, userstream.UserSignupResponseStreamName, responseMsg); err != nil {
			return fmt.Errorf("failed to publish UserSignupResponse event: %w", err)
		}
	}

	h.logger.Info("HandleUserSignupRequest completed", "message_id", msg.UUID)
	msg.Ack()
	return nil
}

func (h *UserHandlers) HandleUserRoleUpdateRequest(ctx context.Context, msg *message.Message) error {
	h.logger.Info("HandleUserRoleUpdateRequest started", "message_id", msg.UUID, "contextErr", ctx.Err())

	var req userevents.UserRoleUpdateRequestPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal UserRoleUpdateRequest", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal UserRoleUpdateRequest: %w", err)
	}

	// Validate UserRole field
	if !req.NewRole.IsValid() {
		err := fmt.Errorf("invalid UserRole: %s", req.NewRole.String())
		h.logger.Error("Validation failed for UserRoleUpdateRequest", "error", err, "message_id", msg.UUID, "new_role", req.NewRole.String())
		return fmt.Errorf("validation error: %w", err)
	}

	// Set a timeout context for the service call
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()

	resp, err := h.UserService.OnUserRoleUpdateRequest(timeoutCtx, req)
	if err != nil {
		// Log the specific error and a general message
		h.logger.Error("Failed to process user role update request", "error", err, "message_id", msg.UUID, "error_msg", err.Error())
		return fmt.Errorf("failed to process user role update request: %w", err)
	}

	// Publish the response event using the EventBus
	// Use the string literal for the event type
	if err := h.publishEvent(ctx, userevents.UserRoleUpdateResponse, resp); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdateResponse event: %w", err)
	}

	h.logger.Info("HandleUserRoleUpdateRequest completed successfully", "message_id", msg.UUID)
	msg.Ack()
	return nil
}

func (h *UserHandlers) publishEvent(ctx context.Context, subject string, payload interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadData)
	msg.Metadata.Set("subject", subject)

	// Determine the stream name based on the event type
	streamName := userstream.StreamNameForEvent(subject)

	h.logger.Info("Publishing event", "subject", subject, "payload", string(payloadData))

	if err := h.EventBus.Publish(ctx, streamName, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
