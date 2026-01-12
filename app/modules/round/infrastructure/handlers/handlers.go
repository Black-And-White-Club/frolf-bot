package roundhandlers

import (
	"context"
	"log/slog"
	"time"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"go.opentelemetry.io/otel/trace"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	roundService roundservice.Service
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      roundmetrics.RoundMetrics
	helpers      utils.Helpers
}

// NewRoundHandlers creates a new instance of RoundHandlers.
func NewRoundHandlers(
	roundService roundservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics roundmetrics.RoundMetrics,
) Handlers {
	return &RoundHandlers{
		roundService: roundService,
		logger:       logger,
		tracer:       tracer,
		helpers:      helpers,
		metrics:      metrics,
	}
}

// extractAnchorClock builds an AnchorClock from context if a timestamp is provided; falls back to RealClock.
func (h *RoundHandlers) extractAnchorClock(ctx context.Context) roundutil.Clock {
	// Typically, the wrapper or middleware would inject the "submitted_at" time into the context.
	// We check for it here to maintain deterministic parsing.
	if t, ok := ctx.Value("submitted_at").(time.Time); ok {
		return roundutil.NewAnchorClock(t)
	}
	return roundutil.RealClock{}
}
