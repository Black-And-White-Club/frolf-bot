package authhandlers

import (
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"go.opentelemetry.io/otel/trace"
)

// AuthHandlers implements the Handlers interface for auth events.
type AuthHandlers struct {
	service  authservice.Service
	eventBus eventbus.EventBus
	helper   utils.Helpers
	logger   *slog.Logger
	tracer   trace.Tracer
}

// NewAuthHandlers creates a new AuthHandlers instance.
func NewAuthHandlers(
	service authservice.Service,
	eventBus eventbus.EventBus,
	helper utils.Helpers,
	logger *slog.Logger,
	tracer trace.Tracer,
) Handlers {
	return &AuthHandlers{
		service:  service,
		eventBus: eventBus,
		helper:   helper,
		logger:   logger,
		tracer:   tracer,
	}
}
