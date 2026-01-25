package leaderboardhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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

// mapSuccessResults is a private helper to build consistent batch completion events.
func (h *LeaderboardHandlers) mapSuccessResults(
	guildID sharedtypes.GuildID,
	requestorID sharedtypes.DiscordID,
	batchID string,
	result results.OperationResult,
	source sharedtypes.ServiceUpdateSource,
) []handlerwrapper.Result {
	var assignments []leaderboardevents.TagAssignmentInfoV1
	if result.IsSuccess() {
		if payload, ok := result.Success.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1); ok {
			assignments = payload.Assignments
		}
	}

	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, a := range assignments {
		changedTags[a.UserID] = a.TagNumber
	}

	return []handlerwrapper.Result{
		{
			Topic: leaderboardevents.LeaderboardBatchTagAssignedV1,
			Payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID: guildID, RequestingUserID: requestorID, BatchID: batchID,
				AssignmentCount: len(assignments), Assignments: assignments,
			},
		},
		{
			// THIS IS THE TRIGGER
			Topic: sharedevents.SyncRoundsTagRequestV1,
			Payload: &sharedevents.SyncRoundsTagRequestPayloadV1{
				GuildID:     guildID,
				ChangedTags: changedTags,
				UpdatedAt:   time.Now().UTC(),
				Source:      source,
			},
		},
	}
}

// addGuildScopedResult appends a guild-scoped version of the event for PWA permission scoping.
// This enables PWA consumers to subscribe with patterns like "leaderboard.updated.v1.{guild_id}".
// Maintains backward compatibility by keeping the original non-scoped event.
func addGuildScopedResult(results []handlerwrapper.Result, baseTopic string, guildID any) []handlerwrapper.Result {
	// Convert guildID to string
	var guildIDStr string
	switch v := guildID.(type) {
	case string:
		guildIDStr = v
	case fmt.Stringer:
		guildIDStr = v.String()
	default:
		guildIDStr = fmt.Sprintf("%v", v)
	}

	if guildIDStr == "" {
		return results
	}

	// Find the result with the matching base topic and duplicate it with guild suffix
	for _, r := range results {
		if r.Topic == baseTopic {
			guildScopedTopic := fmt.Sprintf("%s.%s", baseTopic, guildIDStr)
			guildScopedResult := handlerwrapper.Result{
				Topic:    guildScopedTopic,
				Payload:  r.Payload,
				Metadata: r.Metadata,
			}
			return append(results, guildScopedResult)
		}
	}

	return results
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
