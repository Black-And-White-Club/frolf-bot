package authhandlers

import (
	"context"
	"encoding/json"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/nats-io/nats.go"
)

// MagicLinkRequest is the incoming request payload for magic link generation
type MagicLinkRequest struct {
	UserID  string `json:"user_id"`
	GuildID string `json:"guild_id"`
	Role    string `json:"role"`
}

// MagicLinkResponse is the response payload sent back to requesters
type MagicLinkResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
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
		resp := MagicLinkResponse{
			Success: false,
			Error:   "invalid request format",
		}
		h.respondMagicLink(msg, resp)
		return
	}

	h.logger.InfoContext(ctx, "Generating magic link",
		attr.String("user_id", req.UserID),
		attr.String("guild_id", req.GuildID),
		attr.String("role", req.Role),
	)

	// Convert role string to domain type
	role := authdomain.Role(req.Role)

	// Delegate to service
	response, err := h.service.GenerateMagicLink(ctx, req.UserID, req.GuildID, role)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to generate magic link",
			attr.Error(err),
			attr.String("user_id", req.UserID),
		)
		resp := MagicLinkResponse{
			Success: false,
			Error:   err.Error(),
		}
		h.respondMagicLink(msg, resp)
		return
	}

	// Convert service response (authservice.MagicLinkResponse) to local response
	resp := MagicLinkResponse{
		Success: response.Success,
		URL:     response.URL,
		Error:   response.Error,
	}
	h.respondMagicLink(msg, resp)

	if response.Success {
		h.logger.InfoContext(ctx, "Magic link generated successfully",
			attr.String("user_id", req.UserID),
			attr.String("guild_id", req.GuildID),
		)
	}
}

// respondMagicLink sends a magic link response.
func (h *AuthHandlers) respondMagicLink(msg *nats.Msg, resp MagicLinkResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal magic link response",
			attr.Error(err),
		)
		return
	}

	if err := msg.Respond(data); err != nil {
		h.logger.Error("Failed to respond to magic link request",
			attr.Error(err),
		)
	}
}
