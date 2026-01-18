package leaderboardhandlers

import (
	"context"
	"log/slog"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardHandlers implements the Handlers interface for leaderboard events.
type LeaderboardHandlers struct {
	service         leaderboardservice.Service
	sagaCoordinator *saga.SwapSagaCoordinator
	helpers         utils.Helpers
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(
	service leaderboardservice.Service,
	sagaCoordinator *saga.SwapSagaCoordinator,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics leaderboardmetrics.LeaderboardMetrics,
) Handlers {
	return &LeaderboardHandlers{
		service:         service,
		sagaCoordinator: sagaCoordinator,
		helpers:         helpers,
	}
}

// mapOperationResult converts a service OperationResult to handler Results.
// For standard single-topic success/failure patterns.
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

// HandleGuildConfigCreated seeds an empty active leaderboard for the guild if missing.
func (h *LeaderboardHandlers) HandleGuildConfigCreated(
	ctx context.Context,
	payload *guildevents.GuildConfigCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if err := h.service.EnsureGuildLeaderboard(ctx, payload.GuildID); err != nil {
		return nil, err
	}
	return []handlerwrapper.Result{}, nil
}
