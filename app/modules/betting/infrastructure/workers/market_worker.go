package bettingworkers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
)

const defaultTickInterval = 10 * time.Minute

// guildDiscoverer enumerates guilds that have upcoming rounds, allowing the
// worker to fan out EnsureMarketsForGuild calls without depending on the full
// round repository interface.
type guildDiscoverer interface {
	DiscoverGuildsWithUpcomingRounds(ctx context.Context, lookahead time.Duration) ([]sharedtypes.GuildID, error)
}

// marketClient is the subset of bettingservice.Service the worker calls.
// Using a narrow interface makes the worker easier to test.
type marketClient interface {
	EnsureMarketsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketGeneratedResult, error)
	LockDueMarkets(ctx context.Context) ([]bettingservice.MarketLockResult, error)
}

// MarketWorker is a background ticker that ensures winner markets exist for
// all entitled guilds' upcoming rounds. It runs on a fixed interval and
// publishes BettingMarketGeneratedV1 events for any newly created markets.
type MarketWorker struct {
	service      marketClient
	guilds       guildDiscoverer
	eventBus     eventbus.EventBus
	helpers      utils.Helpers
	logger       *slog.Logger
	tickInterval time.Duration
	stop         chan struct{}
}

func NewMarketWorker(
	service marketClient,
	guilds guildDiscoverer,
	eventBus eventbus.EventBus,
	helpers utils.Helpers,
	logger *slog.Logger,
) *MarketWorker {
	return &MarketWorker{
		service:      service,
		guilds:       guilds,
		eventBus:     eventBus,
		helpers:      helpers,
		logger:       logger,
		tickInterval: defaultTickInterval,
		stop:         make(chan struct{}),
	}
}

// Start begins the ticker loop. It blocks until ctx is cancelled or Stop is
// called — run it in a goroutine.
func (w *MarketWorker) Start(ctx context.Context) {
	w.logger.InfoContext(ctx, "betting market worker starting",
		attr.String("interval", w.tickInterval.String()),
	)

	// Run once immediately so markets are available right at startup.
	w.tick(ctx)

	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.tick(ctx)
		case <-w.stop:
			w.logger.InfoContext(ctx, "betting market worker stopped")
			return
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "betting market worker context cancelled")
			return
		}
	}
}

// Stop signals the worker to exit gracefully.
func (w *MarketWorker) Stop() {
	select {
	case w.stop <- struct{}{}:
	default:
	}
}

func (w *MarketWorker) tick(ctx context.Context) {
	guildIDs, err := w.guilds.DiscoverGuildsWithUpcomingRounds(ctx, 48*time.Hour)
	if err != nil {
		w.logger.WarnContext(ctx, "betting market worker: failed to discover guilds",
			attr.Error(err),
		)
		return
	}

	for _, guildID := range guildIDs {
		results, err := w.service.EnsureMarketsForGuild(ctx, guildID)
		if err != nil {
			w.logger.WarnContext(ctx, "betting market worker: ensure markets failed",
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			continue
		}
		for _, r := range results {
			w.publishMarketGenerated(ctx, r)
		}
	}

	// Lock any markets whose locks_at has passed, regardless of guild.
	lockResults, err := w.service.LockDueMarkets(ctx)
	if err != nil {
		w.logger.WarnContext(ctx, "betting market worker: lock due markets failed",
			attr.Error(err),
		)
	} else {
		for _, r := range lockResults {
			w.publishMarketLocked(ctx, r)
		}
	}
}

func (w *MarketWorker) publishMarketGenerated(ctx context.Context, r bettingservice.MarketGeneratedResult) {
	payload := bettingevents.BettingMarketGeneratedPayloadV1{
		GuildID:    r.GuildID,
		ClubUUID:   r.ClubUUID,
		RoundID:    r.RoundID,
		MarketID:   r.MarketID,
		MarketType: r.MarketType,
	}
	msg, err := w.helpers.CreateNewMessage(payload, bettingevents.BettingMarketGeneratedV1)
	if err != nil {
		w.logger.WarnContext(ctx, "betting market worker: failed to create message",
			attr.String("guild_id", string(r.GuildID)),
			attr.Error(err),
		)
		return
	}
	if err := w.eventBus.Publish(bettingevents.BettingMarketGeneratedV1, msg); err != nil {
		w.logger.WarnContext(ctx, "betting market worker: failed to publish event",
			attr.String("guild_id", string(r.GuildID)),
			attr.Error(err),
		)
	}
	if r.ClubUUID != "" {
		scopedSubject := fmt.Sprintf("%s.%s", bettingevents.BettingMarketGeneratedV1, r.ClubUUID)
		if err := w.eventBus.Publish(scopedSubject, msg); err != nil {
			w.logger.WarnContext(ctx, "betting market worker: failed to publish club-scoped event",
				attr.String("guild_id", string(r.GuildID)),
				attr.Error(err),
			)
		}
	}
}

func (w *MarketWorker) publishMarketLocked(ctx context.Context, r bettingservice.MarketLockResult) {
	payload := bettingevents.BettingMarketLockedPayloadV1{
		GuildID:  r.GuildID,
		ClubUUID: r.ClubUUID,
		RoundID:  r.RoundID,
		MarketID: r.MarketID,
	}
	msg, err := w.helpers.CreateNewMessage(payload, bettingevents.BettingMarketLockedV1)
	if err != nil {
		w.logger.WarnContext(ctx, "betting market worker: failed to create locked message",
			attr.String("market_id", string(rune(r.MarketID))),
			attr.Error(err),
		)
		return
	}
	if err := w.eventBus.Publish(bettingevents.BettingMarketLockedV1, msg); err != nil {
		w.logger.WarnContext(ctx, "betting market worker: failed to publish locked event",
			attr.String("guild_id", string(r.GuildID)),
			attr.Error(err),
		)
	}
	if r.ClubUUID != "" {
		scopedSubject := fmt.Sprintf("%s.%s", bettingevents.BettingMarketLockedV1, r.ClubUUID)
		if err := w.eventBus.Publish(scopedSubject, msg); err != nil {
			w.logger.WarnContext(ctx, "betting market worker: failed to publish club-scoped locked event",
				attr.String("guild_id", string(r.GuildID)),
				attr.Error(err),
			)
		}
	}
}
