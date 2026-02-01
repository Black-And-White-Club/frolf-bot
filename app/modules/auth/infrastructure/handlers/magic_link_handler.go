package authhandlers

import (
	"context"
	"encoding/json"

	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/nats-io/nats.go"
)

// MagicLinkRequest is the incoming request payload for magic link generation
type MagicLinkRequest struct {
	UserID        string `json:"user_id"`
	GuildID       string `json:"guild_id"`
	Role          string `json:"role"`
	CorrelationID string `json:"correlation_id"`
}

// HandleMagicLinkRequest handles incoming magic link requests via NATS.
func (h *AuthHandlers) HandleMagicLinkRequest(msg *nats.Msg) {
	ctx := context.Background()
	ctx, span := h.tracer.Start(ctx, "AuthHandlers.HandleMagicLinkRequest")
	defer span.End()

	var req MagicLinkRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to unmarshal magic link request",
			attr.Error(err),
		)
		// Cannot reply if we can't parse the request
		return
	}

	h.logger.InfoContext(ctx, "Generating magic link",
		attr.String("user_id", req.UserID),
		attr.String("guild_id", req.GuildID),
		attr.String("role", req.Role),
		attr.String("correlation_id", req.CorrelationID),
	)

	// Convert role string to domain type
	role := authdomain.Role(req.Role)

	// Delegate to service
	response, err := h.service.GenerateMagicLink(ctx, req.UserID, req.GuildID, role)

	// Prepare response payload
	respPayload := authevents.MagicLinkGeneratedPayload{
		UserID:        req.UserID,
		GuildID:       req.GuildID,
		CorrelationID: req.CorrelationID,
		Success:       err == nil,
	}

	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to generate magic link",
			attr.Error(err),
			attr.String("user_id", req.UserID),
		)
		respPayload.Error = err.Error()
	} else {
		respPayload.URL = response.URL
		h.logger.InfoContext(ctx, "Magic link generated successfully",
			attr.String("user_id", req.UserID),
			attr.String("guild_id", req.GuildID),
		)
	}

	// Publish response event
	outMsg, err := h.helper.CreateNewMessage(respPayload, authevents.MagicLinkGeneratedV1)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to create magic link response message", attr.Error(err))
		return
	}

	// Ensure Nats-Msg-Id is unique (Watermill helper does this, but good to be sure)
	// Publish to the event bus
	if err := h.eventBus.Publish(authevents.MagicLinkGeneratedV1, outMsg); err != nil {
		h.logger.ErrorContext(ctx, "Failed to publish magic link response", attr.Error(err))
		return
	}

	h.logger.InfoContext(ctx, "Published magic link response",
		attr.String("correlation_id", req.CorrelationID),
		attr.Bool("success", respPayload.Success),
	)
}
