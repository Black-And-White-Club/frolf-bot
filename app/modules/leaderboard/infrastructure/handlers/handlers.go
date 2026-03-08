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
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// RoundLookup provides read-only round data for cross-module enrichment.
type RoundLookup interface {
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
}

// LeaderboardHandlers implements the Handlers interface for leaderboard events.
type LeaderboardHandlers struct {
	service         leaderboardservice.Service
	userService     userservice.Service
	sagaCoordinator saga.SagaCoordinator
	helpers         utils.Helpers
	logger          *slog.Logger
	roundLookup     RoundLookup
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
	roundLookup RoundLookup,
) Handlers {
	return &LeaderboardHandlers{
		service:         service,
		userService:     userService,
		sagaCoordinator: sagaCoordinator,
		helpers:         helpers,
		logger:          logger,
		roundLookup:     roundLookup,
	}
}

// mapSuccessResults is a private helper to build consistent batch completion events.
func (h *LeaderboardHandlers) mapSuccessResults(
	guildID sharedtypes.GuildID,
	requestorID sharedtypes.DiscordID,
	batchID string,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	replyTo string,
) []handlerwrapper.Result {
	// Build assignments for the success event, excluding removals (TagNumber=0).
	// Removals are not tag assignments; including them causes consumers (e.g. Discord)
	// to treat tag #0 as a real tag.
	assignments := make([]leaderboardevents.TagAssignmentInfoV1, 0, len(requests))
	for _, req := range requests {
		if req.TagNumber == 0 {
			continue
		}
		assignments = append(assignments, leaderboardevents.TagAssignmentInfoV1{
			UserID:    req.UserID,
			TagNumber: req.TagNumber,
		})
	}

	// changedTags for round-sync includes removals (tag=0 means "user lost their tag").
	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, req := range requests {
		changedTags[req.UserID] = req.TagNumber
	}

	topic := leaderboardevents.LeaderboardBatchTagAssignedV2
	if replyTo != "" {
		topic = replyTo
	}

	results := []handlerwrapper.Result{
		{
			Topic: topic,
			Payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID:          guildID,
				RequestingUserID: requestorID,
				BatchID:          batchID,
				AssignmentCount:  len(requests),
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

	// Publish per-user tag update events for PWA live updates.
	guildIDStr := string(guildID)
	for _, req := range requests {
		newTag := req.TagNumber
		scopedTopic := fmt.Sprintf("%s.%s", leaderboardevents.LeaderboardTagUpdatedV2, guildIDStr)
		results = append(results, handlerwrapper.Result{
			Topic: scopedTopic,
			Payload: &leaderboardevents.LeaderboardTagUpdatedPayloadV1{
				GuildID: guildID,
				UserID:  req.UserID,
				NewTag:  &newTag,
				Reason:  string(source),
			},
		})
	}

	return results
}

// addGuildScopedResult appends a guild-scoped version of the event for PWA permission scoping.
// This enables PWA consumers to subscribe with patterns like "leaderboard.updated.v2.{guild_id}".
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
