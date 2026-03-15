package bettingservice

import (
	"context"
	"io"
	"log/slog"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

func enabledEntitlements() guildtypes.ResolvedClubEntitlements {
	return guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: {
				Key:   guildtypes.ClubFeatureBetting,
				State: guildtypes.FeatureAccessStateEnabled,
			},
		},
	}
}

func disabledEntitlements() guildtypes.ResolvedClubEntitlements {
	return guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: {
				Key:   guildtypes.ClubFeatureBetting,
				State: guildtypes.FeatureAccessStateDisabled,
			},
		},
	}
}

func frozenEntitlements() guildtypes.ResolvedClubEntitlements {
	return guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: {
				Key:   guildtypes.ClubFeatureBetting,
				State: guildtypes.FeatureAccessStateFrozen,
			},
		},
	}
}

// adminMembership returns a ClubMembership with Admin role.
func adminMembership(userUUID, clubUUID uuid.UUID) *userdb.ClubMembership {
	return &userdb.ClubMembership{
		UserUUID: userUUID,
		ClubUUID: clubUUID,
		Role:     sharedtypes.UserRoleAdmin,
	}
}

// memberMembership returns a ClubMembership with a regular user role.
func memberMembership(userUUID, clubUUID uuid.UUID) *userdb.ClubMembership {
	return &userdb.ClubMembership{
		UserUUID: userUUID,
		ClubUUID: clubUUID,
		Role:     sharedtypes.UserRoleUser,
	}
}

// ---------------------------------------------------------------------------
// FakeBettingRepository
// ---------------------------------------------------------------------------

type FakeBettingRepository struct {
	trace []string

	GetMemberSettingsFunc         func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*bettingdb.MemberSetting, error)
	UpsertMemberSettingsFunc      func(ctx context.Context, db bun.IDB, setting *bettingdb.MemberSetting) error
	CreateWalletJournalEntryFunc  func(ctx context.Context, db bun.IDB, entry *bettingdb.WalletJournalEntry) error
	GetWalletJournalBalanceFunc   func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	ListWalletJournalFunc         func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, limit int) ([]bettingdb.WalletJournalEntry, error)
	GetReservedStakeTotalFunc     func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	GetMarketByRoundFunc          func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, seasonID string, roundID uuid.UUID, marketType string) (*bettingdb.Market, error)
	GetMarketByIDFunc             func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, marketID int64) (*bettingdb.Market, error)
	ListMarketsByRoundFunc        func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, roundID uuid.UUID) ([]bettingdb.Market, error)
	ListMarketsForClubFunc        func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, limit int) ([]bettingdb.Market, error)
	CreateMarketFunc              func(ctx context.Context, db bun.IDB, market *bettingdb.Market) error
	UpdateMarketFunc              func(ctx context.Context, db bun.IDB, market *bettingdb.Market) error
	SuspendOpenMarketsForClubFunc func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]bettingdb.SuspendedMarketRef, error)
	ListOpenMarketsToLockFunc     func(ctx context.Context, db bun.IDB, now time.Time) ([]bettingdb.Market, error)
	ListMarketOptionsFunc         func(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.MarketOption, error)
	CreateMarketOptionsFunc       func(ctx context.Context, db bun.IDB, options []bettingdb.MarketOption) error
	CreateBetFunc                 func(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error
	UpdateBetFunc                 func(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error
	ListBetsForMarketFunc         func(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.Bet, error)
	ListBetsForUserAndMarketFunc  func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, marketID int64) ([]bettingdb.Bet, error)
	CreateAuditLogFunc            func(ctx context.Context, db bun.IDB, log *bettingdb.AuditLog) error
	AcquireWalletBalanceFunc      func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (*bettingdb.WalletBalance, error)
	ApplyWalletBalanceDeltaFunc   func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, balanceDelta, reservedDelta int) error
}

func NewFakeBettingRepository() *FakeBettingRepository { return &FakeBettingRepository{} }

func (f *FakeBettingRepository) record(step string) { f.trace = append(f.trace, step) }
func (f *FakeBettingRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeBettingRepository) GetMemberSettings(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*bettingdb.MemberSetting, error) {
	f.record("GetMemberSettings")
	if f.GetMemberSettingsFunc != nil {
		return f.GetMemberSettingsFunc(ctx, db, clubUUID, userUUID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) UpsertMemberSettings(ctx context.Context, db bun.IDB, setting *bettingdb.MemberSetting) error {
	f.record("UpsertMemberSettings")
	if f.UpsertMemberSettingsFunc != nil {
		return f.UpsertMemberSettingsFunc(ctx, db, setting)
	}
	return nil
}

func (f *FakeBettingRepository) CreateWalletJournalEntry(ctx context.Context, db bun.IDB, entry *bettingdb.WalletJournalEntry) error {
	f.record("CreateWalletJournalEntry")
	if f.CreateWalletJournalEntryFunc != nil {
		return f.CreateWalletJournalEntryFunc(ctx, db, entry)
	}
	return nil
}

func (f *FakeBettingRepository) GetWalletJournalBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error) {
	f.record("GetWalletJournalBalance")
	if f.GetWalletJournalBalanceFunc != nil {
		return f.GetWalletJournalBalanceFunc(ctx, db, clubUUID, userUUID, seasonID)
	}
	return 0, nil
}

func (f *FakeBettingRepository) ListWalletJournal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, limit int) ([]bettingdb.WalletJournalEntry, error) {
	f.record("ListWalletJournal")
	if f.ListWalletJournalFunc != nil {
		return f.ListWalletJournalFunc(ctx, db, clubUUID, userUUID, seasonID, limit)
	}
	return nil, nil
}

func (f *FakeBettingRepository) GetReservedStakeTotal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error) {
	f.record("GetReservedStakeTotal")
	if f.GetReservedStakeTotalFunc != nil {
		return f.GetReservedStakeTotalFunc(ctx, db, clubUUID, userUUID, seasonID)
	}
	return 0, nil
}

func (f *FakeBettingRepository) GetMarketByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, seasonID string, roundID uuid.UUID, marketType string) (*bettingdb.Market, error) {
	f.record("GetMarketByRound")
	if f.GetMarketByRoundFunc != nil {
		return f.GetMarketByRoundFunc(ctx, db, clubUUID, seasonID, roundID, marketType)
	}
	return nil, nil
}

func (f *FakeBettingRepository) GetMarketByID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, marketID int64) (*bettingdb.Market, error) {
	f.record("GetMarketByID")
	if f.GetMarketByIDFunc != nil {
		return f.GetMarketByIDFunc(ctx, db, clubUUID, marketID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) ListMarketsByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, roundID uuid.UUID) ([]bettingdb.Market, error) {
	f.record("ListMarketsByRound")
	if f.ListMarketsByRoundFunc != nil {
		return f.ListMarketsByRoundFunc(ctx, db, clubUUID, roundID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) ListMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, limit int) ([]bettingdb.Market, error) {
	f.record("ListMarketsForClub")
	if f.ListMarketsForClubFunc != nil {
		return f.ListMarketsForClubFunc(ctx, db, clubUUID, limit)
	}
	return nil, nil
}

func (f *FakeBettingRepository) CreateMarket(ctx context.Context, db bun.IDB, market *bettingdb.Market) error {
	f.record("CreateMarket")
	if f.CreateMarketFunc != nil {
		return f.CreateMarketFunc(ctx, db, market)
	}
	return nil
}

func (f *FakeBettingRepository) UpdateMarket(ctx context.Context, db bun.IDB, market *bettingdb.Market) error {
	f.record("UpdateMarket")
	if f.UpdateMarketFunc != nil {
		return f.UpdateMarketFunc(ctx, db, market)
	}
	return nil
}

func (f *FakeBettingRepository) SuspendOpenMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]bettingdb.SuspendedMarketRef, error) {
	f.record("SuspendOpenMarketsForClub")
	if f.SuspendOpenMarketsForClubFunc != nil {
		return f.SuspendOpenMarketsForClubFunc(ctx, db, clubUUID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) ListOpenMarketsToLock(ctx context.Context, db bun.IDB, now time.Time) ([]bettingdb.Market, error) {
	f.record("ListOpenMarketsToLock")
	if f.ListOpenMarketsToLockFunc != nil {
		return f.ListOpenMarketsToLockFunc(ctx, db, now)
	}
	return nil, nil
}

func (f *FakeBettingRepository) ListMarketOptions(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.MarketOption, error) {
	f.record("ListMarketOptions")
	if f.ListMarketOptionsFunc != nil {
		return f.ListMarketOptionsFunc(ctx, db, marketID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) CreateMarketOptions(ctx context.Context, db bun.IDB, options []bettingdb.MarketOption) error {
	f.record("CreateMarketOptions")
	if f.CreateMarketOptionsFunc != nil {
		return f.CreateMarketOptionsFunc(ctx, db, options)
	}
	return nil
}

func (f *FakeBettingRepository) CreateBet(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error {
	f.record("CreateBet")
	if f.CreateBetFunc != nil {
		return f.CreateBetFunc(ctx, db, bet)
	}
	return nil
}

func (f *FakeBettingRepository) UpdateBet(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error {
	f.record("UpdateBet")
	if f.UpdateBetFunc != nil {
		return f.UpdateBetFunc(ctx, db, bet)
	}
	return nil
}

func (f *FakeBettingRepository) ListBetsForMarket(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.Bet, error) {
	f.record("ListBetsForMarket")
	if f.ListBetsForMarketFunc != nil {
		return f.ListBetsForMarketFunc(ctx, db, marketID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) ListBetsForUserAndMarket(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, marketID int64) ([]bettingdb.Bet, error) {
	f.record("ListBetsForUserAndMarket")
	if f.ListBetsForUserAndMarketFunc != nil {
		return f.ListBetsForUserAndMarketFunc(ctx, db, clubUUID, userUUID, marketID)
	}
	return nil, nil
}

func (f *FakeBettingRepository) CreateAuditLog(ctx context.Context, db bun.IDB, log *bettingdb.AuditLog) error {
	f.record("CreateAuditLog")
	if f.CreateAuditLogFunc != nil {
		return f.CreateAuditLogFunc(ctx, db, log)
	}
	return nil
}

func (f *FakeBettingRepository) AcquireWalletBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (*bettingdb.WalletBalance, error) {
	f.record("AcquireWalletBalance")
	if f.AcquireWalletBalanceFunc != nil {
		return f.AcquireWalletBalanceFunc(ctx, db, clubUUID, userUUID, seasonID)
	}
	return &bettingdb.WalletBalance{}, nil
}

func (f *FakeBettingRepository) ApplyWalletBalanceDelta(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, balanceDelta, reservedDelta int) error {
	f.record("ApplyWalletBalanceDelta")
	if f.ApplyWalletBalanceDeltaFunc != nil {
		return f.ApplyWalletBalanceDeltaFunc(ctx, db, clubUUID, userUUID, seasonID, balanceDelta, reservedDelta)
	}
	return nil
}

var _ bettingRepository = (*FakeBettingRepository)(nil)

// ---------------------------------------------------------------------------
// FakeUserRepository
// ---------------------------------------------------------------------------

type FakeUserRepository struct {
	trace []string

	GetUUIDByDiscordIDFunc          func(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error)
	GetClubMembershipFunc           func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error)
	GetClubUUIDByDiscordGuildIDFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error)
	GetDiscordGuildIDByClubUUIDFunc func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error)
	GetUserByUUIDFunc               func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*userdb.User, error)
}

func NewFakeUserRepository() *FakeUserRepository { return &FakeUserRepository{} }

func (f *FakeUserRepository) record(step string) { f.trace = append(f.trace, step) }
func (f *FakeUserRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeUserRepository) GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	f.record("GetUUIDByDiscordID")
	if f.GetUUIDByDiscordIDFunc != nil {
		return f.GetUUIDByDiscordIDFunc(ctx, db, discordID)
	}
	return uuid.UUID{}, nil
}

func (f *FakeUserRepository) GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
	f.record("GetClubMembership")
	if f.GetClubMembershipFunc != nil {
		return f.GetClubMembershipFunc(ctx, db, userUUID, clubUUID)
	}
	return nil, nil
}

func (f *FakeUserRepository) GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	f.record("GetClubUUIDByDiscordGuildID")
	if f.GetClubUUIDByDiscordGuildIDFunc != nil {
		return f.GetClubUUIDByDiscordGuildIDFunc(ctx, db, guildID)
	}
	return uuid.UUID{}, nil
}

func (f *FakeUserRepository) GetDiscordGuildIDByClubUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error) {
	f.record("GetDiscordGuildIDByClubUUID")
	if f.GetDiscordGuildIDByClubUUIDFunc != nil {
		return f.GetDiscordGuildIDByClubUUIDFunc(ctx, db, clubUUID)
	}
	return "", nil
}

func (f *FakeUserRepository) GetUserByUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*userdb.User, error) {
	f.record("GetUserByUUID")
	if f.GetUserByUUIDFunc != nil {
		return f.GetUserByUUIDFunc(ctx, db, userUUID)
	}
	return nil, nil
}

var _ userRepository = (*FakeUserRepository)(nil)

// ---------------------------------------------------------------------------
// FakeGuildRepository
// ---------------------------------------------------------------------------

type FakeGuildRepository struct {
	trace []string

	ResolveEntitlementsFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error)
}

func NewFakeGuildRepository() *FakeGuildRepository { return &FakeGuildRepository{} }

func (f *FakeGuildRepository) record(step string) { f.trace = append(f.trace, step) }
func (f *FakeGuildRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeGuildRepository) ResolveEntitlements(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
	f.record("ResolveEntitlements")
	if f.ResolveEntitlementsFunc != nil {
		return f.ResolveEntitlementsFunc(ctx, db, guildID)
	}
	return guildtypes.ResolvedClubEntitlements{}, nil
}

var _ guildRepository = (*FakeGuildRepository)(nil)

// ---------------------------------------------------------------------------
// FakeLeaderboardRepository
// ---------------------------------------------------------------------------

type FakeLeaderboardRepository struct {
	trace []string

	GetActiveSeasonFunc   func(ctx context.Context, db bun.IDB, guildID string) (*leaderboarddb.Season, error)
	GetSeasonStandingFunc func(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error)
}

func NewFakeLeaderboardRepository() *FakeLeaderboardRepository { return &FakeLeaderboardRepository{} }

func (f *FakeLeaderboardRepository) record(step string) { f.trace = append(f.trace, step) }
func (f *FakeLeaderboardRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeLeaderboardRepository) GetActiveSeason(ctx context.Context, db bun.IDB, guildID string) (*leaderboarddb.Season, error) {
	f.record("GetActiveSeason")
	if f.GetActiveSeasonFunc != nil {
		return f.GetActiveSeasonFunc(ctx, db, guildID)
	}
	return nil, nil
}

func (f *FakeLeaderboardRepository) GetSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error) {
	f.record("GetSeasonStanding")
	if f.GetSeasonStandingFunc != nil {
		return f.GetSeasonStandingFunc(ctx, db, guildID, memberID)
	}
	return nil, nil
}

var _ leaderboardRepository = (*FakeLeaderboardRepository)(nil)

// ---------------------------------------------------------------------------
// FakeRoundRepository
// ---------------------------------------------------------------------------

type FakeRoundRepository struct {
	trace []string

	GetRoundFunc                     func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	GetUpcomingRoundsFunc            func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)
	GetFinalizedRoundsAfterFunc      func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, startTime time.Time) ([]*roundtypes.Round, error)
	GetAllUpcomingRoundsInWindowFunc func(ctx context.Context, db bun.IDB, lookahead time.Duration) ([]*roundtypes.Round, error)
}

func NewFakeRoundRepository() *FakeRoundRepository { return &FakeRoundRepository{} }

func (f *FakeRoundRepository) record(step string) { f.trace = append(f.trace, step) }
func (f *FakeRoundRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeRoundRepository) GetRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	f.record("GetRound")
	if f.GetRoundFunc != nil {
		return f.GetRoundFunc(ctx, db, guildID, roundID)
	}
	return nil, nil
}

func (f *FakeRoundRepository) GetUpcomingRounds(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	f.record("GetUpcomingRounds")
	if f.GetUpcomingRoundsFunc != nil {
		return f.GetUpcomingRoundsFunc(ctx, db, guildID)
	}
	return nil, nil
}

func (f *FakeRoundRepository) GetFinalizedRoundsAfter(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, startTime time.Time) ([]*roundtypes.Round, error) {
	f.record("GetFinalizedRoundsAfter")
	if f.GetFinalizedRoundsAfterFunc != nil {
		return f.GetFinalizedRoundsAfterFunc(ctx, db, guildID, startTime)
	}
	return nil, nil
}

func (f *FakeRoundRepository) GetAllUpcomingRoundsInWindow(ctx context.Context, db bun.IDB, lookahead time.Duration) ([]*roundtypes.Round, error) {
	f.record("GetAllUpcomingRoundsInWindow")
	if f.GetAllUpcomingRoundsInWindowFunc != nil {
		return f.GetAllUpcomingRoundsInWindowFunc(ctx, db, lookahead)
	}
	return nil, nil
}

var _ roundRepository = (*FakeRoundRepository)(nil)

// ---------------------------------------------------------------------------
// newTestService
// ---------------------------------------------------------------------------

func newTestService(
	repo bettingRepository,
	userRepo userRepository,
	guildRepo guildRepository,
	leaderboardRepo leaderboardRepository,
	roundRepo roundRepository,
) *BettingService {
	if roundRepo == nil {
		roundRepo = NewFakeRoundRepository()
	}
	return NewService(
		repo,
		userRepo,
		guildRepo,
		leaderboardRepo,
		roundRepo,
		bettingmetrics.NewNoop(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		nil,
	)
}
