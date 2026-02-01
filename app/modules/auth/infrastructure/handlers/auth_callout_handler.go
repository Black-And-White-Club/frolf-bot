package authhandlers

import (
	"context"
	"encoding/json"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/nats-io/nats.go"
)

// HandleNATSAuthCallout processes an auth callout message from NATS.
func (h *AuthHandlers) HandleNATSAuthCallout(msg *nats.Msg) {
	ctx, span := h.tracer.Start(context.Background(), "AuthHandlers.HandleNATSAuthCallout")
	defer span.End()

	h.logger.DebugContext(ctx, "Received auth callout request",
		attr.String("subject", msg.Subject),
		attr.Int("data_length", len(msg.Data)),
	)

	// Parse the auth request
	var req authservice.NATSAuthRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to unmarshal auth request",
			attr.Error(err),
		)
		h.respondWithError(msg, "invalid request format")
		return
	}

	// Process the auth request
	resp, err := h.service.HandleNATSAuthRequest(ctx, &req)
	if err != nil {
		h.logger.ErrorContext(ctx, "Auth request processing failed",
			attr.Error(err),
		)
		h.respondWithError(msg, "internal error")
		return
	}

	// Send response
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to marshal auth response",
			attr.Error(err),
		)
		h.respondWithError(msg, "internal error")
		return
	}

	if err := msg.Respond(respData); err != nil {
		h.logger.ErrorContext(ctx, "Failed to send auth response",
			attr.Error(err),
		)
		return
	}

	if resp.Error != "" {
		h.logger.WarnContext(ctx, "Auth request denied",
			attr.String("error", resp.Error),
		)
	} else {
		h.logger.InfoContext(ctx, "Auth request approved")
	}
}

// respondWithError sends an error response.
func (h *AuthHandlers) respondWithError(msg *nats.Msg, errMsg string) {
	resp := authservice.NATSAuthResponse{
		Error: errMsg,
	}
	respData, _ := json.Marshal(resp)
	if err := msg.Respond(respData); err != nil {
		h.logger.Error("Failed to send error response",
			attr.Error(err),
		)
	}
}
