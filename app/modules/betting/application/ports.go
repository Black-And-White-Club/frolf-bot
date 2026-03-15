package bettingservice

import (
	"context"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// Outbound port interfaces (dependencies injected by the module)
// ---------------------------------------------------------------------------

type bettingRepository interface {
	GetMemberSettings(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*bettingdb.MemberSetting, error)
	UpsertMemberSettings(ctx context.Context, db bun.IDB, setting *bettingdb.MemberSetting) error
	CreateWalletJournalEntry(ctx context.Context, db bun.IDB, entry *bettingdb.WalletJournalEntry) error
	GetWalletJournalBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	ListWalletJournal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, limit int) ([]bettingdb.WalletJournalEntry, error)
	GetReservedStakeTotal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	GetMarketByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, seasonID string, roundID uuid.UUID, marketType string) (*bettingdb.Market, error)
	GetMarketByID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, marketID int64) (*bettingdb.Market, error)
	ListMarketsByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, roundID uuid.UUID) ([]bettingdb.Market, error)
	ListMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, limit int) ([]bettingdb.Market, error)
	CreateMarket(ctx context.Context, db bun.IDB, market *bettingdb.Market) error
	UpdateMarket(ctx context.Context, db bun.IDB, market *bettingdb.Market) error
	SuspendOpenMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]bettingdb.SuspendedMarketRef, error)
	ListOpenMarketsToLock(ctx context.Context, db bun.IDB, now time.Time) ([]bettingdb.Market, error)
	ListMarketOptions(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.MarketOption, error)
	CreateMarketOptions(ctx context.Context, db bun.IDB, options []bettingdb.MarketOption) error
	CreateBet(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error
	UpdateBet(ctx context.Context, db bun.IDB, bet *bettingdb.Bet) error
	ListBetsForMarket(ctx context.Context, db bun.IDB, marketID int64) ([]bettingdb.Bet, error)
	ListBetsForUserAndMarket(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, marketID int64) ([]bettingdb.Bet, error)
	CreateAuditLog(ctx context.Context, db bun.IDB, log *bettingdb.AuditLog) error
	AcquireWalletBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (*bettingdb.WalletBalance, error)
	ApplyWalletBalanceDelta(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, balanceDelta, reservedDelta int) error
}

type userRepository interface {
	GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error)
	GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error)
	GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error)
	GetDiscordGuildIDByClubUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error)
	GetUserByUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*userdb.User, error)
}

type guildRepository interface {
	ResolveEntitlements(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error)
}

type leaderboardRepository interface {
	GetActiveSeason(ctx context.Context, db bun.IDB, guildID string) (*leaderboarddb.Season, error)
	GetSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error)
}

type roundRepository interface {
	GetRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	GetUpcomingRounds(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)
	GetFinalizedRoundsAfter(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, startTime time.Time) ([]*roundtypes.Round, error)
	GetAllUpcomingRoundsInWindow(ctx context.Context, db bun.IDB, lookahead time.Duration) ([]*roundtypes.Round, error)
}

// ---------------------------------------------------------------------------
// Internal value types used across multiple files in this package
// ---------------------------------------------------------------------------

type resolvedWallet struct {
	seasonID       string
	seasonName     string
	seasonPoints   int
	bettingBalance int
	reserved       int
}

type targetParticipant struct {
	participant roundtypes.Participant
	userUUID    uuid.UUID
	label       string
}

type pricedOption struct {
	optionKey        string
	memberID         sharedtypes.DiscordID
	label            string
	probabilityBps   int
	decimalOddsCents int
	metadata         string // JSON metadata stored with the option (e.g. {"line":52} for O/U)
}

type settlementDecision struct {
	status  string
	payout  int
	reason  string
	summary string
}

type winnerOutcome struct {
	status             string
	resolvedOptionKeys string
	summary            string
	voidReason         string
	winners            map[string]struct{}
	scratched          map[string]struct{}
}
