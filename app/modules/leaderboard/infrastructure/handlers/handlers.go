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
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardHandlers implements the Handlers interface for leaderboard events.
type LeaderboardHandlers struct {
	service         leaderboardservice.Service
	userService     userservice.Service
	sagaCoordinator saga.SagaCoordinator
	helpers         utils.Helpers
	logger          *slog.Logger
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(
	service leaderboardservice.Service,
	userService userservice.Service,
	sagaCoordinator saga.SagaCoordinator,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics leaderboardmetrics.LeaderboardMetrics,
) Handlers {
	return &LeaderboardHandlers{
		service:         service,
		userService:     userService,
		sagaCoordinator: sagaCoordinator,
		helpers:         helpers,
		logger:          logger,
	}
}

// mapSuccessResults is a private helper to build consistent batch completion events.
func (h *LeaderboardHandlers) mapSuccessResults(
	guildID sharedtypes.GuildID,
	requestorID sharedtypes.DiscordID,
	batchID string,
	resultData leaderboardtypes.LeaderboardData,
	source sharedtypes.ServiceUpdateSource,
) []handlerwrapper.Result {
	assignments := make([]leaderboardevents.TagAssignmentInfoV1, 0, len(resultData))
	for _, entry := range resultData {
		assignments = append(assignments, leaderboardevents.TagAssignmentInfoV1{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		})
	}

	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, a := range assignments {
		changedTags[a.UserID] = a.TagNumber
	}

	return []handlerwrapper.Result{
		{
			Topic: leaderboardevents.LeaderboardBatchTagAssignedV1,
			Payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID:          guildID,
				RequestingUserID: requestorID,
				BatchID:          batchID,
				AssignmentCount:  len(assignments),
				Assignments:      assignments,
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

// addParallelIdentityResults appends both legacy GuildID and internal ClubUUID scoped versions of the event.
func (h *LeaderboardHandlers) addParallelIdentityResults(ctx context.Context, results []handlerwrapper.Result, baseTopic string, guildID sharedtypes.GuildID) []handlerwrapper.Result {
	// 1. Add legacy GuildID scoped result
	results = addGuildScopedResult(results, baseTopic, guildID)

	// 2. Add internal ClubUUID scoped result
	if guildID != "" {
		clubUUID, err := h.userService.GetClubUUIDByDiscordGuildID(ctx, guildID)
		if err == nil && clubUUID != uuid.Nil {
			results = addGuildScopedResult(results, baseTopic, clubUUID)
		}
	}

	return results
}

// HandleGuildConfigCreated seeds an empty active leaderboard for the guild if missing.
func (h *LeaderboardHandlers) HandleGuildConfigCreated(
	ctx context.Context,
	payload *guildevents.GuildConfigCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.EnsureGuildLeaderboard(ctx, payload.GuildID)
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}
	return []handlerwrapper.Result{}, nil
}
