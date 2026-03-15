package bettingworkers

import (
	"context"
	"sync"
	"testing"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
)

// ---- stubs ----

type fakeGuildDiscoverer struct {
	guildIDs []sharedtypes.GuildID
}

func (f *fakeGuildDiscoverer) DiscoverGuildsWithUpcomingRounds(_ context.Context, _ time.Duration) ([]sharedtypes.GuildID, error) {
	return f.guildIDs, nil
}

// fakeMarketClient satisfies the narrow marketClient interface.
type fakeMarketClient struct {
	mu          sync.Mutex
	called      []sharedtypes.GuildID
	results     map[sharedtypes.GuildID][]bettingservice.MarketGeneratedResult
	lockResults []bettingservice.MarketLockResult
	lockErr     error
}

func (s *fakeMarketClient) EnsureMarketsForGuild(_ context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketGeneratedResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.called = append(s.called, guildID)
	return s.results[guildID], nil
}

func (s *fakeMarketClient) LockDueMarkets(_ context.Context) ([]bettingservice.MarketLockResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lockResults, s.lockErr
}

// ---- tests ----

func TestMarketWorker_Tick_CallsEnsureForDiscoveredGuilds(t *testing.T) {
	guilds := &fakeGuildDiscoverer{
		guildIDs: []sharedtypes.GuildID{"guild-alpha", "guild-beta"},
	}
	svc := &fakeMarketClient{}

	w := &MarketWorker{
		service:      svc,
		guilds:       guilds,
		tickInterval: defaultTickInterval,
		stop:         make(chan struct{}),
	}

	w.tick(context.Background())

	svc.mu.Lock()
	defer svc.mu.Unlock()

	if len(svc.called) != 2 {
		t.Fatalf("expected EnsureMarketsForGuild called twice, got %d", len(svc.called))
	}
	set := make(map[sharedtypes.GuildID]bool, 2)
	for _, g := range svc.called {
		set[g] = true
	}
	for _, want := range []sharedtypes.GuildID{"guild-alpha", "guild-beta"} {
		if !set[want] {
			t.Errorf("expected EnsureMarketsForGuild called for %s", want)
		}
	}
}

func TestMarketWorker_Tick_EmptyGuilds_NoCall(t *testing.T) {
	guilds := &fakeGuildDiscoverer{guildIDs: nil}
	svc := &fakeMarketClient{}

	w := &MarketWorker{
		service:      svc,
		guilds:       guilds,
		tickInterval: defaultTickInterval,
		stop:         make(chan struct{}),
	}

	w.tick(context.Background())

	svc.mu.Lock()
	defer svc.mu.Unlock()

	if len(svc.called) != 0 {
		t.Errorf("expected no calls, got %d", len(svc.called))
	}
}

// TestMarketWorker_Tick_LocksDueMarkets verifies that LockDueMarkets is invoked
// on every tick. Publishing is covered by integration tests; here we just
// confirm the worker calls through to the service.
func TestMarketWorker_Tick_LocksDueMarkets(t *testing.T) {
	t.Parallel()

	guilds := &fakeGuildDiscoverer{guildIDs: nil}
	var lockCalled bool
	svc := &fakeMarketClient{
		lockResults: []bettingservice.MarketLockResult{
			{GuildID: "guild-1", ClubUUID: "club-uuid-1", MarketID: 99},
		},
	}
	_ = svc // suppress unused

	client := &fakeMarketClient{}
	client.lockResults = []bettingservice.MarketLockResult{{GuildID: "guild-1", MarketID: 99}}

	// Wrap to track the call without a real eventbus.
	tracking := &trackingMarketClient{inner: client, onLock: func() { lockCalled = true }}

	w := &MarketWorker{
		service:      tracking,
		guilds:       guilds,
		eventBus:     nil,
		helpers:      nil,
		logger:       nil,
		tickInterval: defaultTickInterval,
		stop:         make(chan struct{}),
	}

	w.tick(context.Background())

	if !lockCalled {
		t.Error("expected LockDueMarkets to be called")
	}
}

// trackingMarketClient wraps fakeMarketClient to intercept LockDueMarkets.
type trackingMarketClient struct {
	inner  *fakeMarketClient
	onLock func()
}

func (t *trackingMarketClient) EnsureMarketsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketGeneratedResult, error) {
	return t.inner.EnsureMarketsForGuild(ctx, guildID)
}

func (t *trackingMarketClient) LockDueMarkets(ctx context.Context) ([]bettingservice.MarketLockResult, error) {
	if t.onLock != nil {
		t.onLock()
	}
	// Return nil to avoid triggering publishMarketLocked (no eventbus in unit tests).
	return nil, nil
}
