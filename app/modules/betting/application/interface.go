package bettingservice

import (
	"context"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

type Service interface {
	GetOverview(ctx context.Context, clubUUID, userUUID uuid.UUID) (*Overview, error)
	GetNextRoundMarket(ctx context.Context, clubUUID, userUUID uuid.UUID) (*NextRoundMarket, error)
	GetMarketSnapshot(ctx context.Context, clubUUID uuid.UUID) (*MarketSnapshot, error)
	GetAdminMarkets(ctx context.Context, clubUUID, userUUID uuid.UUID) (*AdminMarketBoard, error)
	UpdateSettings(ctx context.Context, req UpdateSettingsRequest) (*MemberSettings, error)
	AdjustWallet(ctx context.Context, req AdjustWalletRequest) (*WalletJournal, error)
	PlaceBet(ctx context.Context, req PlaceBetRequest) (*BetTicket, error)
	AdminMarketAction(ctx context.Context, req AdminMarketActionRequest) (*AdminMarketActionResult, error)
	SettleRound(ctx context.Context, guildID sharedtypes.GuildID, round *BettingSettlementRound, source string, actorUUID *uuid.UUID, reason string) ([]MarketSettlementResult, error)
	VoidRoundMarkets(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, source string, actorUUID *uuid.UUID, reason string) ([]MarketVoidResult, error)
	// EnsureMarketsForGuild generates or reprices winner markets for all upcoming
	// rounds belonging to the given guild. Used by the background market worker.
	EnsureMarketsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]MarketGeneratedResult, error)
	// LockDueMarkets transitions all open markets whose locks_at has passed to
	// locked status. Returns results for the caller to emit domain events.
	LockDueMarkets(ctx context.Context) ([]MarketLockResult, error)
	// SuspendOpenMarketsForClub suspends all open markets for a club in response
	// to a feature access change (freeze or disable). Accepted bets on suspended
	// markets remain valid and will settle normally.
	SuspendOpenMarketsForClub(ctx context.Context, guildID sharedtypes.GuildID) ([]MarketSuspendedResult, error)
	// MirrorPointsToWallet journals season-point deltas from a round into the
	// betting wallet for each awarded player. Idempotent: a given roundID is
	// only journaled once per user per season (enforced by DB unique index).
	MirrorPointsToWallet(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, points map[sharedtypes.DiscordID]int) error
}

type Overview struct {
	ClubUUID     string          `json:"club_uuid"`
	GuildID      string          `json:"guild_id"`
	SeasonID     string          `json:"season_id"`
	SeasonName   string          `json:"season_name"`
	AccessState  string          `json:"access_state"`
	AccessSource string          `json:"access_source"`
	ReadOnly     bool            `json:"read_only"`
	Wallet       WalletSnapshot  `json:"wallet"`
	Settings     MemberSettings  `json:"settings"`
	Journal      []WalletJournal `json:"journal"`
}

type WalletSnapshot struct {
	SeasonPoints      int `json:"season_points"`
	AdjustmentBalance int `json:"adjustment_balance"`
	Available         int `json:"available"`
	Reserved          int `json:"reserved"`
}

type MemberSettings struct {
	OptOutTargeting bool      `json:"opt_out_targeting"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WalletJournal struct {
	ID        int64     `json:"id"`
	EntryType string    `json:"entry_type"`
	Amount    int       `json:"amount"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type NextRoundMarket struct {
	ClubUUID    string          `json:"club_uuid"`
	GuildID     string          `json:"guild_id"`
	SeasonID    string          `json:"season_id"`
	AccessState string          `json:"access_state"`
	ReadOnly    bool            `json:"read_only"`
	Wallet      WalletSnapshot  `json:"wallet"`
	Round       BettingRound    `json:"round"`
	Market      *BettingMarket  `json:"market,omitempty"` // compat: first market (winner)
	Markets     []BettingMarket `json:"markets"`          // all markets for this round
	UserBets    []BetTicket     `json:"user_bets"`
	Warnings    []string        `json:"warnings"`
}

type BettingRound struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
}

type BettingMarket struct {
	ID        int64                 `json:"id"`
	Type      string                `json:"type"`
	Title     string                `json:"title"`
	Status    string                `json:"status"`
	LocksAt   time.Time             `json:"locks_at"`
	Ephemeral bool                  `json:"ephemeral"`
	Result    string                `json:"result,omitempty"`
	Options   []BettingMarketOption `json:"options"`
}

type BettingMarketOption struct {
	OptionKey          string  `json:"option_key"`
	MemberID           string  `json:"member_id"`
	Label              string  `json:"label"`
	ProbabilityPercent int     `json:"probability_percent"`
	DecimalOdds        float64 `json:"decimal_odds"`
	Metadata           string  `json:"metadata,omitempty"`
	SelfBetRestricted  bool    `json:"self_bet_restricted,omitempty"`
}

type BetTicket struct {
	ID              int64      `json:"id"`
	RoundID         string     `json:"round_id"`
	MarketType      string     `json:"market_type"`
	SelectionKey    string     `json:"selection_key"`
	SelectionLabel  string     `json:"selection_label"`
	Stake           int        `json:"stake"`
	DecimalOdds     float64    `json:"decimal_odds"`
	PotentialPayout int        `json:"potential_payout"`
	SettledPayout   int        `json:"settled_payout"`
	Status          string     `json:"status"`
	SettledAt       *time.Time `json:"settled_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AdminMarketBoard struct {
	ClubUUID string               `json:"club_uuid"`
	GuildID  string               `json:"guild_id"`
	Markets  []AdminMarketSummary `json:"markets"`
}

type AdminMarketSummary struct {
	ID                int64      `json:"id"`
	RoundID           string     `json:"round_id"`
	RoundTitle        string     `json:"round_title"`
	MarketType        string     `json:"market_type"`
	Title             string     `json:"title"`
	Status            string     `json:"status"`
	LocksAt           time.Time  `json:"locks_at"`
	SettledAt         *time.Time `json:"settled_at"`
	ResultSummary     string     `json:"result_summary"`
	SettlementVersion int        `json:"settlement_version"`
	TicketCount       int        `json:"ticket_count"`
	Exposure          int        `json:"exposure"`
	AcceptedTickets   int        `json:"accepted_tickets"`
	WonTickets        int        `json:"won_tickets"`
	LostTickets       int        `json:"lost_tickets"`
	VoidedTickets     int        `json:"voided_tickets"`
}

type UpdateSettingsRequest struct {
	ClubUUID        uuid.UUID `json:"club_uuid"`
	UserUUID        uuid.UUID `json:"-"`
	OptOutTargeting bool      `json:"opt_out_targeting"`
}

type AdjustWalletRequest struct {
	ClubUUID  uuid.UUID             `json:"club_uuid"`
	AdminUUID uuid.UUID             `json:"-"`
	MemberID  sharedtypes.DiscordID `json:"member_id"`
	Amount    int                   `json:"amount"`
	Reason    string                `json:"reason"`
}

type PlaceBetRequest struct {
	ClubUUID       uuid.UUID           `json:"club_uuid"`
	UserUUID       uuid.UUID           `json:"-"`
	RoundID        sharedtypes.RoundID `json:"round_id"`
	MarketType     string              `json:"market_type,omitempty"` // defaults to winnerMarketType if empty
	SelectionKey   string              `json:"selection_key"`
	Stake          int                 `json:"stake"`
	IdempotencyKey string              `json:"idempotency_key,omitempty"` // optional; empty means no idempotency protection
}

type AdminMarketActionRequest struct {
	ClubUUID  uuid.UUID `json:"club_uuid"`
	AdminUUID uuid.UUID `json:"-"`
	MarketID  int64     `json:"market_id"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
}

type AdminMarketActionResult struct {
	MarketID          int64      `json:"market_id"`
	Action            string     `json:"action"`
	Status            string     `json:"status"`
	ResultSummary     string     `json:"result_summary"`
	SettlementVersion int        `json:"settlement_version"`
	SettledAt         *time.Time `json:"settled_at"`
	AffectedMarketIDs []int64    `json:"affected_market_ids,omitempty"`
}

type BettingSettlementRound struct {
	ID           sharedtypes.RoundID
	Title        string
	GuildID      sharedtypes.GuildID
	Finalized    bool
	Participants []BettingSettlementParticipant
}

type BettingSettlementParticipant struct {
	MemberID string
	Response string
	Score    *int
	IsDNF    bool
}

// MarketSettlementResult carries the outcome of settling one market, returned
// by SettleRound so the event handler can emit betting.market.settled events.
type MarketSettlementResult struct {
	GuildID           sharedtypes.GuildID
	ClubUUID          string
	RoundID           sharedtypes.RoundID
	MarketID          int64
	ResultSummary     string
	SettlementVersion int
}

// MarketVoidResult carries the outcome of voiding one market, returned by
// VoidRoundMarkets so the event handler can emit betting.market.voided events.
type MarketVoidResult struct {
	GuildID  sharedtypes.GuildID
	ClubUUID string
	RoundID  sharedtypes.RoundID
	MarketID int64
	Reason   string
}

// MarketLockResult carries the outcome of locking one market, returned by
// LockDueMarkets so the background worker can emit betting.market.locked events.
type MarketLockResult struct {
	GuildID  sharedtypes.GuildID
	ClubUUID string
	RoundID  sharedtypes.RoundID
	MarketID int64
}

// MarketSnapshot is the public (non-user-specific) view of the next upcoming
// betting market for a club. It contains no wallet data or per-user bets —
// those remain behind HTTP auth. Used by the NATS snapshot request/reply.
type MarketSnapshot struct {
	ClubUUID    string          `json:"club_uuid"`
	GuildID     string          `json:"guild_id"`
	SeasonID    string          `json:"season_id"`
	AccessState string          `json:"access_state"`
	Round       *BettingRound   `json:"round,omitempty"`
	Market      *BettingMarket  `json:"market,omitempty"`  // compat: first market
	Markets     []BettingMarket `json:"markets,omitempty"` // all markets for this round
}
type MarketGeneratedResult struct {
	GuildID    sharedtypes.GuildID
	ClubUUID   string
	RoundID    sharedtypes.RoundID
	MarketID   int64
	MarketType string
}

// MarketSuspendedResult carries the ID of a market suspended due to an
// entitlement loss event. Used by the event handler to emit domain events.
type MarketSuspendedResult struct {
	GuildID  sharedtypes.GuildID
	ClubUUID string
	RoundID  sharedtypes.RoundID
	MarketID int64
}
