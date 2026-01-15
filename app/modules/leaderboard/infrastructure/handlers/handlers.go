package leaderboardhandlers

import (
	"context"
	"log/slog"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	leaderboardService leaderboardservice.Service
	sagaCoordinator    *saga.SwapSagaCoordinator
	logger             *slog.Logger
	tracer             trace.Tracer
	metrics            leaderboardmetrics.LeaderboardMetrics
	helpers            utils.Helpers
}

// NewLeaderboardHandlers creates a new instance of LeaderboardHandlers.
func NewLeaderboardHandlers(
	leaderboardService leaderboardservice.Service,
	sagaCoordinator *saga.SwapSagaCoordinator,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics leaderboardmetrics.LeaderboardMetrics,
) Handlers {
	return &LeaderboardHandlers{
		leaderboardService: leaderboardService,
		sagaCoordinator:    sagaCoordinator,
		logger:             logger,
		tracer:             tracer,
		helpers:            helpers,
		metrics:            metrics,
	}
}

// HandleGuildConfigCreated seeds an empty active leaderboard for the guild if missing.
func (h *LeaderboardHandlers) HandleGuildConfigCreated(
	ctx context.Context,
	payload *guildevents.GuildConfigCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received GuildConfigCreated event",
		attr.ExtractCorrelationID(ctx),
		attr.String("guild_id", string(payload.GuildID)),
	)

	if err := h.leaderboardService.EnsureGuildLeaderboard(ctx, payload.GuildID); err != nil {
		return nil, err
	}

	h.logger.InfoContext(ctx, "Guild leaderboard ensured",
		attr.ExtractCorrelationID(ctx),
	)

	return []handlerwrapper.Result{}, nil
}
