package bettingintegrationtests

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"

	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"

	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

var (
	testEnv     *testutils.TestEnvironment
	testEnvErr  error
	testEnvOnce sync.Once

	standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score", "betting"}
)

// BettingTestDeps holds all dependencies for a single betting service integration test.
type BettingTestDeps struct {
	Ctx     context.Context
	DB      bettingdb.Repository
	BunDB   *bun.DB
	Service bettingservice.Service
	Cleanup func()
}

// BettingWorld is the seeded test world returned by SeedBettingWorld.
type BettingWorld struct {
	GuildID         sharedtypes.GuildID
	ClubUUID        uuid.UUID
	SeasonID        string
	MemberDiscordID sharedtypes.DiscordID
	MemberUUID      uuid.UUID
	AdminDiscordID  sharedtypes.DiscordID
	AdminUUID       uuid.UUID
}

// GetTestEnv returns (or lazily initialises) the shared TestEnvironment.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initialising betting application test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Betting application test environment initialised successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Betting test environment initialisation failed: %v", testEnvErr)
	}
	if testEnv == nil {
		t.Fatalf("Betting test environment not initialised")
	}
	return testEnv
}

// SetupTestBettingService resets the environment and constructs a real BettingService backed by
// real repositories.
func SetupTestBettingService(t *testing.T) BettingTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noOpTracer := noop.NewTracerProvider().Tracer("test_betting_service")

	repo := bettingdb.NewRepository(env.DB)
	userRepo := userdb.NewRepository(env.DB)
	guildRepo := guilddb.NewRepository(env.DB)
	leaderboardRepo := leaderboarddb.NewRepository(env.DB)
	roundRepo := rounddb.NewRepository(env.DB)

	service := bettingservice.NewService(
		repo,
		userRepo,
		guildRepo,
		leaderboardRepo,
		roundRepo,
		bettingmetrics.NewNoop(),
		testLogger,
		noOpTracer,
		env.DB,
	)

	testCtx, testCancel := context.WithCancel(env.Ctx)

	cleanup := func() { testCancel() }
	t.Cleanup(cleanup)

	return BettingTestDeps{
		Ctx:     testCtx,
		DB:      repo,
		BunDB:   env.DB,
		Service: service,
		Cleanup: cleanup,
	}
}

// SeedBettingWorld inserts the minimum set of rows required for most betting service
// calls: a club, guild config (trial enabled), two users (member + admin), their
// club memberships, and an active leaderboard season.
func SeedBettingWorld(t *testing.T, db *bun.DB) BettingWorld {
	t.Helper()

	ctx := context.Background()

	guildIDStr := "tbg-" + uuid.New().String()[:8]
	guildID := sharedtypes.GuildID(guildIDStr)
	clubUUID := uuid.New()
	memberDiscordID := sharedtypes.DiscordID("member-" + uuid.New().String()[:8])
	memberUUID := uuid.New()
	adminDiscordID := sharedtypes.DiscordID("admin-" + uuid.New().String()[:8])
	adminUUID := uuid.New()
	seasonID := "2026-test"

	// 1. Club row — needed by GetClubUUIDByDiscordGuildID / GetDiscordGuildIDByClubUUID.
	club := &clubdb.Club{
		UUID:           clubUUID,
		Name:           "Test Betting Club",
		DiscordGuildID: &guildIDStr,
	}
	if _, err := db.NewInsert().Model(club).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert club: %v", err)
	}

	// 2. Guild config with is_trial=true so betting entitlement resolves as Enabled.
	guildCfg := &guilddb.GuildConfig{
		GuildID:  guildID,
		IsTrial:  true,
		IsActive: true,
	}
	if _, err := db.NewInsert().Model(guildCfg).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert guild_configs: %v", err)
	}

	// 3. Global user rows.
	memberUser := &userdb.User{UUID: memberUUID, UserID: &memberDiscordID}
	if _, err := db.NewInsert().Model(memberUser).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert member user: %v", err)
	}
	adminUser := &userdb.User{UUID: adminUUID, UserID: &adminDiscordID}
	if _, err := db.NewInsert().Model(adminUser).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert admin user: %v", err)
	}

	// 4. Club memberships so GetClubMembership works.
	memberCM := &userdb.ClubMembership{
		UserUUID: memberUUID,
		ClubUUID: clubUUID,
		Role:     sharedtypes.UserRoleUser,
	}
	if _, err := db.NewInsert().Model(memberCM).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert member club_membership: %v", err)
	}
	adminCM := &userdb.ClubMembership{
		UserUUID: adminUUID,
		ClubUUID: clubUUID,
		Role:     sharedtypes.UserRoleAdmin,
	}
	if _, err := db.NewInsert().Model(adminCM).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert admin club_membership: %v", err)
	}

	// 5. Active leaderboard season.
	season := &leaderboarddb.Season{
		GuildID:  guildIDStr,
		ID:       seasonID,
		Name:     "Test Season 2026",
		IsActive: true,
	}
	if _, err := db.NewInsert().Model(season).Exec(ctx); err != nil {
		t.Fatalf("SeedBettingWorld: insert leaderboard_season: %v", err)
	}

	return BettingWorld{
		GuildID:         guildID,
		ClubUUID:        clubUUID,
		SeasonID:        seasonID,
		MemberDiscordID: memberDiscordID,
		MemberUUID:      memberUUID,
		AdminDiscordID:  adminDiscordID,
		AdminUUID:       adminUUID,
	}
}

// SeedRound inserts an upcoming round with optional accepted participants and returns its ID.
// Pass participants to allow EnsureMarketsForGuild to generate markets (needs ≥ 2 accepted).
func SeedRound(t *testing.T, db *bun.DB, roundRepo rounddb.Repository, guildID sharedtypes.GuildID, participants ...roundtypes.Participant) sharedtypes.RoundID {
	t.Helper()

	roundID := sharedtypes.RoundID(uuid.New())
	startTime := sharedtypes.StartTime(time.Now().Add(48 * time.Hour).UTC())

	round := &roundtypes.Round{
		ID:           roundID,
		GuildID:      guildID,
		Title:        "Test Round",
		StartTime:    &startTime,
		State:        roundtypes.RoundStateUpcoming,
		Participants: participants,
	}
	if err := roundRepo.CreateRound(context.Background(), db, guildID, round); err != nil {
		t.Fatalf("SeedRound: create round: %v", err)
	}
	return roundID
}

// SeedMarketWithBet creates an open market with one option and one accepted bet.
// Returns the market ID and bet ID.
func SeedMarketWithBet(
	t *testing.T,
	db *bun.DB,
	repo bettingdb.Repository,
	world BettingWorld,
	roundID sharedtypes.RoundID,
) (marketID int64, betID int64) {
	t.Helper()

	ctx := context.Background()

	market := &bettingdb.Market{
		ClubUUID:   world.ClubUUID,
		SeasonID:   world.SeasonID,
		RoundID:    uuid.UUID(roundID),
		MarketType: "winner",
		Title:      "Who wins?",
		Status:     "open",
		LocksAt:    time.Now().Add(24 * time.Hour),
	}
	if err := repo.CreateMarket(ctx, db, market); err != nil {
		t.Fatalf("SeedMarketWithBet: create market: %v", err)
	}

	optionKey := string(world.MemberDiscordID)
	opts := []bettingdb.MarketOption{
		{
			MarketID:            market.ID,
			OptionKey:           optionKey,
			ParticipantMemberID: string(world.MemberDiscordID),
			Label:               "Member",
			ProbabilityBps:      5000,
			DecimalOddsCents:    200,
			DisplayOrder:        1,
		},
	}
	if err := repo.CreateMarketOptions(ctx, db, opts); err != nil {
		t.Fatalf("SeedMarketWithBet: create options: %v", err)
	}

	bet := &bettingdb.Bet{
		ClubUUID:         world.ClubUUID,
		UserUUID:         world.MemberUUID,
		SeasonID:         world.SeasonID,
		RoundID:          uuid.UUID(roundID),
		MarketID:         market.ID,
		MarketType:       "winner",
		SelectionKey:     optionKey,
		SelectionLabel:   "Member",
		Stake:            50,
		DecimalOddsCents: 200,
		PotentialPayout:  100,
		Status:           "accepted",
	}
	if err := repo.CreateBet(ctx, db, bet); err != nil {
		t.Fatalf("SeedMarketWithBet: create bet: %v", err)
	}

	return market.ID, bet.ID
}
