package bettingdb

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MemberSetting struct {
	bun.BaseModel `bun:"table:betting_member_settings,alias:bms"`

	ClubUUID        uuid.UUID `bun:"club_uuid,pk,type:uuid,notnull"`
	UserUUID        uuid.UUID `bun:"user_uuid,pk,type:uuid,notnull"`
	OptOutTargeting bool      `bun:"opt_out_targeting,notnull,default:false"`
	UpdatedAt       time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type WalletJournalEntry struct {
	bun.BaseModel `bun:"table:betting_wallet_journal,alias:bwj"`

	ID            int64      `bun:"id,pk,autoincrement"`
	ClubUUID      uuid.UUID  `bun:"club_uuid,type:uuid,notnull"`
	UserUUID      uuid.UUID  `bun:"user_uuid,type:uuid,notnull"`
	SeasonID      string     `bun:"season_id,type:varchar(64),notnull"`
	EntryType     string     `bun:"entry_type,type:varchar(64),notnull"`
	Amount        int        `bun:"amount,notnull"`
	Reason        string     `bun:"reason,type:text,notnull,default:''"`
	CreatedBy     string     `bun:"created_by,type:varchar(128),notnull,default:''"`
	SourceRoundID *uuid.UUID `bun:"source_round_id,type:uuid,nullzero"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type Market struct {
	bun.BaseModel `bun:"table:betting_markets,alias:bm"`

	ID                int64      `bun:"id,pk,autoincrement"`
	ClubUUID          uuid.UUID  `bun:"club_uuid,type:uuid,notnull"`
	SeasonID          string     `bun:"season_id,type:varchar(64),notnull"`
	RoundID           uuid.UUID  `bun:"round_id,type:uuid,notnull"`
	MarketType        string     `bun:"market_type,type:varchar(64),notnull"`
	Title             string     `bun:"title,type:text,notnull"`
	Status            string     `bun:"status,type:varchar(32),notnull"`
	LocksAt           time.Time  `bun:"locks_at,nullzero,notnull"`
	ResolvedOptionKey string     `bun:"resolved_option_key,type:text,notnull,default:''"`
	VoidReason        string     `bun:"void_reason,type:text,notnull,default:''"`
	ResultSummary     string     `bun:"result_summary,type:text,notnull,default:''"`
	SettlementVersion int        `bun:"settlement_version,notnull,default:0"`
	LastResultSource  string     `bun:"last_result_source,type:varchar(128),notnull,default:''"`
	SettledAt         *time.Time `bun:"settled_at,nullzero"`
	CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt         time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type MarketOption struct {
	bun.BaseModel `bun:"table:betting_market_options,alias:bmo"`

	ID                  int64  `bun:"id,pk,autoincrement"`
	MarketID            int64  `bun:"market_id,notnull"`
	OptionKey           string `bun:"option_key,type:varchar(128),notnull"`
	ParticipantMemberID string `bun:"participant_member_id,type:varchar(32),notnull"`
	Label               string `bun:"label,type:text,notnull"`
	ProbabilityBps      int    `bun:"probability_bps,notnull"`
	DecimalOddsCents    int    `bun:"decimal_odds_cents,notnull"`
	DisplayOrder        int    `bun:"display_order,notnull"`
	Metadata            string `bun:"metadata,type:text,notnull,default:''"`
}

type Bet struct {
	bun.BaseModel `bun:"table:betting_bets,alias:bb"`

	ID               int64      `bun:"id,pk,autoincrement"`
	ClubUUID         uuid.UUID  `bun:"club_uuid,type:uuid,notnull"`
	UserUUID         uuid.UUID  `bun:"user_uuid,type:uuid,notnull"`
	SeasonID         string     `bun:"season_id,type:varchar(64),notnull"`
	RoundID          uuid.UUID  `bun:"round_id,type:uuid,notnull"`
	MarketID         int64      `bun:"market_id,notnull"`
	MarketType       string     `bun:"market_type,type:varchar(64),notnull"`
	SelectionKey     string     `bun:"selection_key,type:varchar(128),notnull"`
	SelectionLabel   string     `bun:"selection_label,type:text,notnull"`
	Stake            int        `bun:"stake,notnull"`
	DecimalOddsCents int        `bun:"decimal_odds_cents,notnull"`
	PotentialPayout  int        `bun:"potential_payout,notnull"`
	SettledPayout    int        `bun:"settled_payout,notnull,default:0"`
	Status           string     `bun:"status,type:varchar(32),notnull"`
	IdempotencyKey   *string    `bun:"idempotency_key,type:varchar(128),nullzero"`
	SettledAt        *time.Time `bun:"settled_at,nullzero"`
	CreatedAt        time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// WalletBalance is a denormalized projection of a user's betting wallet for a
// given club and season. It tracks the current betting-journal balance and the
// total stake currently reserved in accepted bets. It is the authoritative
// source for balance checks in PlaceBet: callers must acquire it with
// SELECT FOR UPDATE (AcquireWalletBalance) before reading or modifying it.
// Season points from the leaderboard are NOT stored here — they are read live
// and added to Balance-Reserved at check time.
type WalletBalance struct {
	bun.BaseModel `bun:"table:betting_wallet_balances,alias:bwb"`

	ClubUUID uuid.UUID `bun:"club_uuid,pk,type:uuid,notnull"`
	UserUUID uuid.UUID `bun:"user_uuid,pk,type:uuid,notnull"`
	SeasonID string    `bun:"season_id,pk,type:varchar(64),notnull"`
	Balance  int       `bun:"balance,notnull,default:0"`
	Reserved int       `bun:"reserved,notnull,default:0"`
}

type AuditLog struct {
	bun.BaseModel `bun:"table:betting_audit_log,alias:bal"`

	ID            int64      `bun:"id,pk,autoincrement"`
	ClubUUID      uuid.UUID  `bun:"club_uuid,type:uuid,notnull"`
	MarketID      *int64     `bun:"market_id"`
	RoundID       *uuid.UUID `bun:"round_id,type:uuid"`
	ActorUserUUID *uuid.UUID `bun:"actor_user_uuid,type:uuid"`
	Action        string     `bun:"action,type:varchar(64),notnull"`
	Reason        string     `bun:"reason,type:text,notnull,default:''"`
	Metadata      string     `bun:"metadata,type:text,notnull,default:''"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}
