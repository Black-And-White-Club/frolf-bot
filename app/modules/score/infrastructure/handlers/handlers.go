package scorehandlers

import (
	"log/slog"

	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"go.opentelemetry.io/otel/trace"
)

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	scoreService scoreservice.Service
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      scoremetrics.ScoreMetrics
	Helpers      utils.Helpers
}

// NewScoreHandlers creates a new ScoreHandlers.
func NewScoreHandlers(
	scoreService scoreservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics scoremetrics.ScoreMetrics,
) Handlers {
	return &ScoreHandlers{
		scoreService: scoreService,
		logger:       logger,
		tracer:       tracer,
		Helpers:      helpers,
		metrics:      metrics,
	}
}
