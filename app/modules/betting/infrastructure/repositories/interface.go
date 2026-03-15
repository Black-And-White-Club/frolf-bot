package bettingdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository interface {
	GetMemberSettings(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*MemberSetting, error)
	UpsertMemberSettings(ctx context.Context, db bun.IDB, setting *MemberSetting) error
	CreateWalletJournalEntry(ctx context.Context, db bun.IDB, entry *WalletJournalEntry) error
	GetWalletJournalBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	ListWalletJournal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, limit int) ([]WalletJournalEntry, error)
	GetReservedStakeTotal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error)
	GetMarketByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, seasonID string, roundID uuid.UUID, marketType string) (*Market, error)
	GetMarketByID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, marketID int64) (*Market, error)
	ListMarketsByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, roundID uuid.UUID) ([]Market, error)
	ListMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, limit int) ([]Market, error)
	ListOpenMarketsToLock(ctx context.Context, db bun.IDB, now time.Time) ([]Market, error)
	CreateMarket(ctx context.Context, db bun.IDB, market *Market) error
	UpdateMarket(ctx context.Context, db bun.IDB, market *Market) error
	// SuspendOpenMarketsForClub atomically transitions all open markets for the
	// given club to suspended status. Returns the IDs of affected markets.
	SuspendOpenMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]SuspendedMarketRef, error)
	ListMarketOptions(ctx context.Context, db bun.IDB, marketID int64) ([]MarketOption, error)
	CreateMarketOptions(ctx context.Context, db bun.IDB, options []MarketOption) error
	CreateBet(ctx context.Context, db bun.IDB, bet *Bet) error
	UpdateBet(ctx context.Context, db bun.IDB, bet *Bet) error
	ListBetsForMarket(ctx context.Context, db bun.IDB, marketID int64) ([]Bet, error)
	ListBetsForUserAndMarket(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, marketID int64) ([]Bet, error)
	CreateAuditLog(ctx context.Context, db bun.IDB, log *AuditLog) error
	// AcquireWalletBalance ensures a balance row exists for (club, user, season)
	// and returns it under a SELECT FOR UPDATE lock. Must be called within a tx.
	AcquireWalletBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (*WalletBalance, error)
	// ApplyWalletBalanceDelta atomically adds balanceDelta to balance and
	// reservedDelta to reserved, creating the row if it does not yet exist.
	ApplyWalletBalanceDelta(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, balanceDelta, reservedDelta int) error
}
