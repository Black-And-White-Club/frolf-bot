package userhandlers

import (
	"log/slog"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"go.opentelemetry.io/otel/trace"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	userService userservice.Service
	logger      *slog.Logger
	tracer      trace.Tracer
	metrics     usermetrics.UserMetrics
	helpers     utils.Helpers
}

// NewUserHandlers creates a new UserHandlers.
func NewUserHandlers(
	userService userservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics usermetrics.UserMetrics,
) Handlers {
	return &UserHandlers{
		userService: userService,
		logger:      logger,
		tracer:      tracer,
		helpers:     helpers,
		metrics:     metrics,
	}
}
