package roundhandlers

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	RoundService roundservice.Service
	logger       observability.Logger
	tracer       observability.Tracer
	helpers      utils.Helpers
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(roundService roundservice.Service, logger observability.Logger, tracer observability.Tracer, helpers utils.Helpers) Handlers {
	return &RoundHandlers{
		RoundService: roundService,
		logger:       logger,
		tracer:       tracer,
		helpers:      helpers,
	}
}
