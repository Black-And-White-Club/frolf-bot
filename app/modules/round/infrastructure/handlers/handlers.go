package roundhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	handlerwrapper "github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/google/uuid"
)

// RoundHandlers implements the Handlers interface for round events.
type RoundHandlers struct {
	service     roundservice.Service
	userService userservice.Service
	logger      *slog.Logger
	helpers     utils.Helpers
}

// NewRoundHandlers creates a new RoundHandlers instance.
func NewRoundHandlers(
	service roundservice.Service,
	userService userservice.Service,
	logger *slog.Logger,
	helpers utils.Helpers,
) Handlers {
	return &RoundHandlers{
		service:     service,
		userService: userService,
		logger:      logger,
		helpers:     helpers,
	}
}

// mapOperationResult converts a service OperationResult to handler Results.
func mapOperationResult[S any, F any](
	result results.OperationResult[S, F],
	successTopic, failureTopic string,
) []handlerwrapper.Result {
	handlerResults := result.ToHandlerResults(successTopic, failureTopic)

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

// addGuildScopedResult appends a guild-scoped version of the event for PWA permission scoping.
// This enables PWA consumers to subscribe with patterns like "round.created.v1.{guild_id}".
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
// This allows the PWA to transition to UUIDs while the Discord bot continues using GuildIDs.
func (h *RoundHandlers) addParallelIdentityResults(ctx context.Context, results []handlerwrapper.Result, baseTopic string, guildID sharedtypes.GuildID) []handlerwrapper.Result {
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

// extractAnchorClock builds an AnchorClock from context if a timestamp is provided; falls back to RealClock.
func (h *RoundHandlers) extractAnchorClock(ctx context.Context) roundutil.Clock {
	// Typically, the wrapper or middleware would inject the "submitted_at" time into the context.
	// We check for it here to maintain deterministic parsing.
	if t, ok := ctx.Value("submitted_at").(time.Time); ok {
		return roundutil.NewAnchorClock(t)
	}
	return roundutil.RealClock{}
}
