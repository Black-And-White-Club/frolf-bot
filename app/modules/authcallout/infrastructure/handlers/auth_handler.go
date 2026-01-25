package handlers

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authcallout "github.com/Black-And-White-Club/frolf-bot/app/modules/authcallout/application"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

// AuthHandler handles NATS auth callout messages.
type AuthHandler struct {
	service authcallout.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	service authcallout.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) *AuthHandler {
	return &AuthHandler{
		service: service,
		logger:  logger,
		tracer:  tracer,
	}
}

// HandleAuthCallout processes an auth callout message from NATS.
func (h *AuthHandler) HandleAuthCallout(msg *nats.Msg) {
	ctx, span := h.tracer.Start(context.Background(), "AuthHandler.HandleAuthCallout")
	defer span.End()

	h.logger.DebugContext(ctx, "Received auth callout request",
		attr.String("subject", msg.Subject),
		attr.Int("data_length", len(msg.Data)),
	)

	// Parse the auth request
	var req authcallout.AuthRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to unmarshal auth request",
			attr.Error(err),
		)
		h.respondWithError(msg, "invalid request format")
		return
	}

	// Process the auth request
	resp, err := h.service.HandleAuthRequest(ctx, &req)
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
func (h *AuthHandler) respondWithError(msg *nats.Msg, errMsg string) {
	resp := authcallout.AuthResponse{
		Error: errMsg,
	}
	respData, _ := json.Marshal(resp)
	if err := msg.Respond(respData); err != nil {
		h.logger.Error("Failed to send error response",
			attr.Error(err),
		)
	}
}
