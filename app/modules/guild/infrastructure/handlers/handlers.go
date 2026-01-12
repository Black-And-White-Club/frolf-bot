package guildhandlers

import (
	"log/slog"

	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"go.opentelemetry.io/otel/trace"
)

// GuildHandlers handles guild-related events following the pure transformation pattern.
type GuildHandlers struct {
	guildService guildservice.Service
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      guildmetrics.GuildMetrics
	helpers      utils.Helpers
}

// NewGuildHandlers creates a new instance of GuildHandlers.
func NewGuildHandlers(
	guildService guildservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics guildmetrics.GuildMetrics,
) *GuildHandlers {
	return &GuildHandlers{
		guildService: guildService,
		logger:       logger,
		tracer:       tracer,
		metrics:      metrics,
		helpers:      helpers,
	}
}
