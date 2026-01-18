package roundhandlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	handlerwrapper "github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// RoundHandlers implements the Handlers interface for round events.
type RoundHandlers struct {
	service roundservice.Service
	logger  *slog.Logger
	helpers utils.Helpers
}

// NewRoundHandlers creates a new RoundHandlers instance.
func NewRoundHandlers(
	service roundservice.Service,
	logger *slog.Logger,
	helpers utils.Helpers,
) Handlers {
	return &RoundHandlers{
		service: service,
		logger:  logger,
		helpers: helpers,
	}
}

// mapOperationResult converts a service OperationResult to handler Results.
func mapOperationResult(
	result results.OperationResult,
	successTopic, failureTopic string,
) []handlerwrapper.Result {
	handlerResults := result.MapToHandlerResults(successTopic, failureTopic)

	wrapperResults := make([]handlerwrapper.Result, len(handlerResults))
	for i, hr := range handlerResults {
		wrapperResults[i] = handlerwrapper.Result{
			Topic:    hr.Topic,
			Payload:  hr.Payload,
			Metadata: hr.Metadata,
		}
	}

	return wrapperResults
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
