package bettinghandlerintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/betting"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace/noop"

	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
)

var standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score", "betting"}

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// BettingHandlerTestDeps holds all dependencies for a handler integration test.
type BettingHandlerTestDeps struct {
	*testutils.TestEnvironment
	BettingModule      *betting.Module
	Router             *message.Router
	EventBus           eventbus.EventBus
	ReceivedMsgs       map[string][]*message.Message
	ReceivedMsgsMutex  *sync.Mutex
	PrometheusRegistry *prometheus.Registry
	TestObservability  observability.Observability
	TestHelpers        utils.Helpers
}

// BettingHandlerWorld is the seeded world for handler tests.
type BettingHandlerWorld struct {
	GuildID         sharedtypes.GuildID
	ClubUUID        uuid.UUID
	SeasonID        string
	MemberDiscordID sharedtypes.DiscordID
	MemberUUID      uuid.UUID
	AdminDiscordID  sharedtypes.DiscordID
	AdminUUID       uuid.UUID
}

// GetTestEnv creates or returns the shared TestEnvironment for handler tests.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing betting handler test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Betting handler test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Betting handler test environment initialization failed: %v", testEnvErr)
	}
	if testEnv == nil {
		t.Fatalf("Betting handler test environment not initialized")
	}
	return testEnv
}

// SetupTestBettingHandler resets the environment, wires a full betting.Module (Watermill
// router + event bus), subscribes to output topics for message capture, and registers
// cleanup.
func SetupTestBettingHandler(t *testing.T) BettingHandlerTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	watermillLogger := watermill.NopLogger{}

	eventBusCtx, eventBusCancel := context.WithCancel(env.Ctx)
	routerRunCtx, routerRunCancel := context.WithCancel(env.Ctx)

	eventBusImpl, err := eventbus.NewEventBus(
		eventBusCtx,
		env.Config.NATS.URL,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"backend",
		&eventbusmetrics.NoOpMetrics{},
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		eventBusCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(env.Ctx, streamName); err != nil {
			eventBusImpl.Close()
			eventBusCancel()
			t.Fatalf("Failed to create NATS stream %q: %v", streamName, err)
		}
	}

	routerConfig := message.RouterConfig{CloseTimeout: 2 * time.Second}
	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		Registry: &observability.Registry{
			BettingMetrics: bettingmetrics.NewNoop(),
			ScoreMetrics:   &scoremetrics.NoOpMetrics{},
			Tracer:         noop.NewTracerProvider().Tracer("test"),
			Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}

	realHelpers := utils.NewHelper(slog.New(slog.NewTextHandler(io.Discard, nil)))

	bettingModule, err := betting.NewModule(env.Ctx, betting.ModuleOptions{
		Observability:   testObservability,
		EventBus:        eventBusImpl,
		Router:          watermillRouter,
		Helpers:         realHelpers,
		RouterCtx:       routerRunCtx,
		DB:              env.DB,
		HTTPRouter:      nil,
		UserRepo:        userdb.NewRepository(env.DB),
		GuildRepo:       guilddb.NewRepository(env.DB),
		LeaderboardRepo: leaderboarddb.NewRepository(env.DB),
		RoundRepo:       rounddb.NewRepository(env.DB),
	})
	if err != nil {
		eventBusImpl.Close()
		eventBusCancel()
		routerRunCancel()
		t.Fatalf("Failed to create betting module: %v", err)
	}

	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(routerRunCtx); runErr != nil && runErr != context.Canceled {
			t.Errorf("Watermill router stopped with error: %v", runErr)
		}
	}()

	select {
	case <-watermillRouter.Running():
	case <-time.After(5 * time.Second):
		t.Fatal("router failed to start within 5s")
	}

	cleanup := func() {
		log.Println("Running betting handler test cleanup...")
		if bettingModule != nil {
			if err := bettingModule.Close(); err != nil {
				log.Printf("Error closing betting module: %v", err)
			}
		} else {
			if watermillRouter != nil {
				_ = watermillRouter.Close()
			}
		}
		routerRunCancel()
		eventBusCancel()
		if eventBusImpl != nil {
			_ = eventBusImpl.Close()
		}
		waitCh := make(chan struct{})
		go func() {
			routerWg.Wait()
			close(waitCh)
		}()
		select {
		case <-waitCh:
		case <-time.After(2 * time.Second):
			log.Println("WARNING: betting handler router goroutine wait timed out")
		}
		os.Setenv("APP_ENV", oldEnv)
		log.Println("Betting handler test cleanup finished.")
	}
	t.Cleanup(cleanup)

	localEnv := *env
	localEnv.EventBus = eventBusImpl

	return BettingHandlerTestDeps{
		TestEnvironment:   &localEnv,
		BettingModule:     bettingModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		ReceivedMsgs:      make(map[string][]*message.Message),
		ReceivedMsgsMutex: &sync.Mutex{},
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
	}
}

// SeedBettingHandlerWorld seeds the minimal data required for handler tests.
func SeedBettingHandlerWorld(t *testing.T, db *bun.DB) BettingHandlerWorld {
	t.Helper()
	ctx := context.Background()

	guildID := sharedtypes.GuildID("tgh-" + uuid.New().String()[:8])
	clubUUID := uuid.New()
	memberDiscordID := sharedtypes.DiscordID("member-handler-" + uuid.New().String()[:8])
	memberUUID := uuid.New()
	adminDiscordID := sharedtypes.DiscordID("admin-handler-" + uuid.New().String()[:8])
	adminUUID := uuid.New()
	seasonID := uuid.New().String()

	// clubs
	club := &clubdb.Club{UUID: clubUUID, Name: "Handler Test Club", DiscordGuildID: (*string)(func() *string { s := string(guildID); return &s }())}
	if _, err := db.NewInsert().Model(club).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingHandlerWorld: insert club: %v", err)
	}
	// guild_configs
	guildCfg := &guilddb.GuildConfig{
		GuildID:  guildID,
		IsTrial:  true,
		IsActive: true,
	}
	if _, err := db.NewInsert().Model(guildCfg).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingHandlerWorld: insert guild_config: %v", err)
	}
	// users
	for _, u := range []struct {
		id   uuid.UUID
		disc sharedtypes.DiscordID
	}{{memberUUID, memberDiscordID}, {adminUUID, adminDiscordID}} {
		uRow := &userdb.User{UUID: u.id, UserID: &u.disc}
		if _, err := db.NewInsert().Model(uRow).Exec(ctx); err != nil {
			t.Fatalf("SeedBettingHandlerWorld: insert user %s: %v", u.disc, err)
		}
	}
	// club_memberships
	type membershipRole string
	for _, m := range []struct {
		userUUID uuid.UUID
		role     string
	}{{memberUUID, "User"}, {adminUUID, "Admin"}} {
		mem := &userdb.ClubMembership{UserUUID: m.userUUID, ClubUUID: clubUUID, Role: sharedtypes.UserRoleEnum(m.role)}
		if _, err := db.NewInsert().Model(mem).Exec(ctx); err != nil {
			t.Fatalf("SeedBettingHandlerWorld: insert membership: %v", err)
		}
	}
	// leaderboard_seasons
	season := &leaderboarddb.Season{
		GuildID:  string(guildID),
		ID:       seasonID,
		Name:     "Handler Test Season",
		IsActive: true,
	}
	if _, err := db.NewInsert().Model(season).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingHandlerWorld: insert season: %v", err)
	}

	return BettingHandlerWorld{
		GuildID:         guildID,
		ClubUUID:        clubUUID,
		SeasonID:        seasonID,
		MemberDiscordID: memberDiscordID,
		MemberUUID:      memberUUID,
		AdminDiscordID:  adminDiscordID,
		AdminUUID:       adminUUID,
	}
}
