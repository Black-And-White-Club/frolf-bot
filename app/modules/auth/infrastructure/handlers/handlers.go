package authhandlers

import (
	"log/slog"

	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"go.opentelemetry.io/otel/trace"
)

// AuthHandlers implements the Handlers interface for auth events.
type AuthHandlers struct {
	service authservice.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

// NewAuthHandlers creates a new AuthHandlers instance.
func NewAuthHandlers(
	service authservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) Handlers {
	return &AuthHandlers{
		service: service,
		logger:  logger,
		tracer:  tracer,
	}
}
