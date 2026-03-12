package clubintegrationtests

import (
	"context"
	"database/sql"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"
	"time"

	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	clubqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/queue"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

type ClubTestDeps struct {
	Ctx         context.Context
	BunDB       *bun.DB
	ClubRepo    clubdb.Repository
	UserRepo    userdb.Repository
	Service     clubservice.Service
	Queue       clubqueue.QueueService
	TagReader   *IntegrationTagReader
	RoundReader *IntegrationRoundReader
	Cleanup     func()
}

type IntegrationTagReader struct {
	mu   sync.RWMutex
	rows []leaderboardservice.MemberTagView
}

func (r *IntegrationTagReader) SetRows(rows []leaderboardservice.MemberTagView) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append([]leaderboardservice.MemberTagView(nil), rows...)
}

func (r *IntegrationTagReader) GetTagList(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := make([]leaderboardservice.MemberTagView, 0, len(r.rows))
	for _, row := range r.rows {
		rows = append(rows, leaderboardservice.MemberTagView{
			MemberID: row.MemberID,
			Tag:      cloneIntegrationInt(row.Tag),
		})
	}
	return rows, nil
}

type IntegrationRoundReader struct {
	mu     sync.RWMutex
	rounds map[uuid.UUID]*roundtypes.Round
}

func NewIntegrationRoundReader() *IntegrationRoundReader {
	return &IntegrationRoundReader{rounds: make(map[uuid.UUID]*roundtypes.Round)}
}

func (r *IntegrationRoundReader) SetRound(round *roundtypes.Round) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rounds[uuid.UUID(round.ID)] = cloneIntegrationRound(round)
}

func (r *IntegrationRoundReader) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	round, ok := r.rounds[uuid.UUID(roundID)]
	if !ok {
		return results.FailureResult[*roundtypes.Round, error](sql.ErrNoRows), nil
	}
	return results.SuccessResult[*roundtypes.Round, error](cloneIntegrationRound(round)), nil
}

func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	testEnvOnce.Do(func() {
		log.Println("Initializing club test environment...")
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = err
			log.Printf("Failed to set up test environment: %v", err)
		} else {
			log.Println("Club test environment initialized successfully.")
			testEnv = env
		}
	})

	if testEnvErr != nil {
		t.Fatalf("Club test environment initialization failed: %v", testEnvErr)
	}
	if testEnv == nil {
		t.Fatalf("Club test environment not initialized")
	}

	return testEnv
}

func SetupTestClubService(t *testing.T) ClubTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	if err := env.Reset(resetCtx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueService, err := clubqueue.NewService(
		env.Ctx,
		env.DB,
		testLogger,
		env.Config.Postgres.DSN,
		clubmetrics.NewNoop(),
		env.EventBus,
		utils.NewHelper(testLogger),
	)
	if err != nil {
		t.Fatalf("Failed to create club queue service: %v", err)
	}
	if err := queueService.Start(env.Ctx); err != nil {
		t.Fatalf("Failed to start club queue service: %v", err)
	}

	tagReader := &IntegrationTagReader{}
	roundReader := NewIntegrationRoundReader()
	clubRepo := clubdb.NewRepository(env.DB)
	userRepo := userdb.NewRepository(env.DB)

	service := clubservice.NewClubService(
		clubRepo,
		userRepo,
		queueService,
		tagReader,
		roundReader,
		testLogger,
		clubmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("club-integration-test"),
		env.DB,
	)

	cleanup := func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if queueService != nil {
			_ = queueService.Stop(stopCtx)
		}
	}

	t.Cleanup(cleanup)

	return ClubTestDeps{
		Ctx:         env.Ctx,
		BunDB:       env.DB,
		ClubRepo:    clubRepo,
		UserRepo:    userRepo,
		Service:     service,
		Queue:       queueService,
		TagReader:   tagReader,
		RoundReader: roundReader,
		Cleanup:     cleanup,
	}
}

func seedClub(t *testing.T, deps ClubTestDeps, withDiscord bool) *clubdb.Club {
	t.Helper()

	now := time.Now().UTC()
	club := &clubdb.Club{
		UUID:      uuid.New(),
		Name:      "Integration Club",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if withDiscord {
		guildID := "guild-integration"
		club.DiscordGuildID = &guildID
	}

	_, err := deps.BunDB.NewInsert().Model(club).Exec(deps.Ctx)
	requireNoError(t, err)
	return club
}

func seedClubMember(t *testing.T, deps ClubTestDeps, clubUUID uuid.UUID, externalID string, role sharedtypes.UserRoleEnum) *userdb.ClubMembership {
	t.Helper()

	now := time.Now().UTC()
	userID := sharedtypes.DiscordID("discord-" + externalID)
	user := &userdb.User{
		UserID:    &userID,
		UUID:      uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := deps.BunDB.NewInsert().Model(user).Exec(deps.Ctx)
	requireNoError(t, err)

	membership := &userdb.ClubMembership{
		ID:         uuid.New(),
		UserUUID:   user.UUID,
		ClubUUID:   clubUUID,
		Role:       role,
		Source:     "integration",
		ExternalID: &externalID,
		JoinedAt:   now,
		UpdatedAt:  now,
	}
	_, err = deps.BunDB.NewInsert().Model(membership).Exec(deps.Ctx)
	requireNoError(t, err)
	return membership
}

func countPendingChallengeJobs(t *testing.T, deps ClubTestDeps, challengeID uuid.UUID, kind string) int {
	t.Helper()

	var count int
	err := deps.BunDB.NewSelect().
		Table("river_job").
		ColumnExpr("COUNT(*)").
		Where("args->>'challenge_id' = ?", challengeID.String()).
		Where("kind = ?", kind).
		Where("state IN ('available', 'scheduled')").
		Scan(deps.Ctx, &count)
	requireNoError(t, err)
	return count
}

func loadChallenge(t *testing.T, deps ClubTestDeps, challengeID uuid.UUID) *clubdb.ClubChallenge {
	t.Helper()

	challenge, err := deps.ClubRepo.GetChallengeByUUID(deps.Ctx, deps.BunDB, challengeID)
	requireNoError(t, err)
	return challenge
}

func loadActiveChallengeLink(t *testing.T, deps ClubTestDeps, challengeID uuid.UUID) *clubdb.ClubChallengeRoundLink {
	t.Helper()

	link, err := deps.ClubRepo.GetActiveChallengeRoundLink(deps.Ctx, deps.BunDB, challengeID)
	if err == nil {
		return link
	}
	if err == clubdb.ErrNotFound {
		return nil
	}
	t.Fatalf("failed to load active challenge link: %v", err)
	return nil
}

func updateChallengeExpiry(t *testing.T, deps ClubTestDeps, challengeID uuid.UUID, column string, expiresAt time.Time) {
	t.Helper()

	_, err := deps.BunDB.NewUpdate().
		Table("club_challenges").
		Set(column+" = ?", expiresAt).
		Where("uuid = ?", challengeID).
		Exec(deps.Ctx)
	requireNoError(t, err)
}

func cloneIntegrationInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneIntegrationRound(round *roundtypes.Round) *roundtypes.Round {
	if round == nil {
		return nil
	}
	copyRound := *round
	return &copyRound
}

func intPtr(value int) *int {
	return &value
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
