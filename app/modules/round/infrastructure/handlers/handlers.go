package roundhandlers

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	RoundService roundservice.Service
	logger       observability.Logger
	tracer       observability.Tracer
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(roundService roundservice.Service, logger observability.Logger, tracer observability.Tracer) Handlers {
	return &RoundHandlers{
		RoundService: roundService,
		logger:       logger,
		tracer:       tracer,
	}
}
