package bettingdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Impl struct {
	db bun.IDB
}

func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

func (r *Impl) GetMemberSettings(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*MemberSetting, error) {
	if db == nil {
		db = r.db
	}

	setting := new(MemberSetting)
	err := db.NewSelect().
		Model(setting).
		Where("club_uuid = ?", clubUUID).
		Where("user_uuid = ?", userUUID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bettingdb.GetMemberSettings: %w", err)
	}

	return setting, nil
}

func (r *Impl) UpsertMemberSettings(ctx context.Context, db bun.IDB, setting *MemberSetting) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().
		Model(setting).
		On("CONFLICT (club_uuid, user_uuid) DO UPDATE").
		Set("opt_out_targeting = EXCLUDED.opt_out_targeting").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.UpsertMemberSettings: %w", err)
	}

	return nil
}

func (r *Impl) CreateWalletJournalEntry(ctx context.Context, db bun.IDB, entry *WalletJournalEntry) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().Model(entry).Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.CreateWalletJournalEntry: %w", err)
	}

	return nil
}

func (r *Impl) GetWalletJournalBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error) {
	if db == nil {
		db = r.db
	}

	var result struct {
		Balance int `bun:"balance"`
	}
	if err := db.NewSelect().
		TableExpr("betting_wallet_journal AS bwj").
		ColumnExpr("COALESCE(SUM(bwj.amount), 0) AS balance").
		Where("bwj.club_uuid = ?", clubUUID).
		Where("bwj.user_uuid = ?", userUUID).
		Where("bwj.season_id = ?", seasonID).
		Where("bwj.entry_type != ?", "stake_reserved").
		Scan(ctx, &result); err != nil {
		return 0, fmt.Errorf("bettingdb.GetWalletJournalBalance: %w", err)
	}

	return result.Balance, nil
}

func (r *Impl) ListWalletJournal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, limit int) ([]WalletJournalEntry, error) {
	if db == nil {
		db = r.db
	}

	if limit <= 0 {
		limit = 20
	}

	entries := make([]WalletJournalEntry, 0, limit)
	if err := db.NewSelect().
		Model(&entries).
		Where("club_uuid = ?", clubUUID).
		Where("user_uuid = ?", userUUID).
		Where("season_id = ?", seasonID).
		OrderExpr("created_at DESC, id DESC").
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListWalletJournal: %w", err)
	}

	return entries, nil
}

func (r *Impl) GetReservedStakeTotal(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (int, error) {
	if db == nil {
		db = r.db
	}

	var result struct {
		Reserved int `bun:"reserved"`
	}
	if err := db.NewSelect().
		TableExpr("betting_bets AS bb").
		ColumnExpr("COALESCE(SUM(bb.stake), 0) AS reserved").
		Where("bb.club_uuid = ?", clubUUID).
		Where("bb.user_uuid = ?", userUUID).
		Where("bb.season_id = ?", seasonID).
		Where("bb.status = ?", "accepted").
		Scan(ctx, &result); err != nil {
		return 0, fmt.Errorf("bettingdb.GetReservedStakeTotal: %w", err)
	}

	return result.Reserved, nil
}

func (r *Impl) GetMarketByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, seasonID string, roundID uuid.UUID, marketType string) (*Market, error) {
	if db == nil {
		db = r.db
	}

	market := new(Market)
	err := db.NewSelect().
		Model(market).
		Where("club_uuid = ?", clubUUID).
		Where("season_id = ?", seasonID).
		Where("round_id = ?", roundID).
		Where("market_type = ?", marketType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bettingdb.GetMarketByRound: %w", err)
	}

	return market, nil
}

func (r *Impl) GetMarketByID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, marketID int64) (*Market, error) {
	if db == nil {
		db = r.db
	}

	market := new(Market)
	err := db.NewSelect().
		Model(market).
		Where("id = ?", marketID).
		Where("club_uuid = ?", clubUUID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bettingdb.GetMarketByID: %w", err)
	}

	return market, nil
}

func (r *Impl) ListMarketsByRound(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, roundID uuid.UUID) ([]Market, error) {
	if db == nil {
		db = r.db
	}

	markets := make([]Market, 0, 4)
	if err := db.NewSelect().
		Model(&markets).
		Where("club_uuid = ?", clubUUID).
		Where("round_id = ?", roundID).
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListMarketsByRound: %w", err)
	}

	return markets, nil
}

func (r *Impl) ListMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, limit int) ([]Market, error) {
	if db == nil {
		db = r.db
	}
	if limit <= 0 {
		limit = 25
	}

	markets := make([]Market, 0, limit)
	if err := db.NewSelect().
		Model(&markets).
		Where("club_uuid = ?", clubUUID).
		OrderExpr("created_at DESC, id DESC").
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListMarketsForClub: %w", err)
	}

	return markets, nil
}

func (r *Impl) CreateMarket(ctx context.Context, db bun.IDB, market *Market) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().Model(market).Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.CreateMarket: %w", err)
	}

	return nil
}

func (r *Impl) UpdateMarket(ctx context.Context, db bun.IDB, market *Market) error {
	if db == nil {
		db = r.db
	}

	q := db.NewUpdate().
		Model(market).
		Column(
			"status",
			"resolved_option_key",
			"void_reason",
			"result_summary",
			"settlement_version",
			"last_result_source",
			"settled_at",
			"updated_at",
		).
		WherePK()

	// When SettlementVersion > 0, the caller has already incremented it.
	// Guard against concurrent settlement by requiring the previous version.
	if market.SettlementVersion > 0 {
		q = q.Where("settlement_version = ?", market.SettlementVersion-1)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("bettingdb.UpdateMarket: %w", err)
	}
	if market.SettlementVersion > 0 {
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrSettlementVersionConflict
		}
	}

	return nil
}

// SuspendedMarketRef holds the minimal info returned by SuspendOpenMarketsForClub.
type SuspendedMarketRef struct {
	ID      int64
	RoundID uuid.UUID
}

func (r *Impl) SuspendOpenMarketsForClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]SuspendedMarketRef, error) {
	if db == nil {
		db = r.db
	}

	type row struct {
		ID      int64     `bun:"id"`
		RoundID uuid.UUID `bun:"round_id"`
	}
	var rows []row
	if err := db.NewRaw(
		`UPDATE betting_markets
		 SET status = 'suspended', updated_at = NOW()
		 WHERE club_uuid = ? AND status = 'open'
		 RETURNING id, round_id`,
		clubUUID,
	).Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("bettingdb.SuspendOpenMarketsForClub: %w", err)
	}
	refs := make([]SuspendedMarketRef, len(rows))
	for i, r := range rows {
		refs[i] = SuspendedMarketRef{ID: r.ID, RoundID: r.RoundID}
	}
	return refs, nil
}

func (r *Impl) ListMarketOptions(ctx context.Context, db bun.IDB, marketID int64) ([]MarketOption, error) {
	if db == nil {
		db = r.db
	}

	options := make([]MarketOption, 0, 8)
	if err := db.NewSelect().
		Model(&options).
		Where("market_id = ?", marketID).
		OrderExpr("display_order ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListMarketOptions: %w", err)
	}

	return options, nil
}

func (r *Impl) CreateMarketOptions(ctx context.Context, db bun.IDB, options []MarketOption) error {
	if len(options) == 0 {
		return nil
	}
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().Model(&options).Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.CreateMarketOptions: %w", err)
	}

	return nil
}

func (r *Impl) CreateBet(ctx context.Context, db bun.IDB, bet *Bet) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().Model(bet).Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.CreateBet: %w", err)
	}

	return nil
}

func (r *Impl) UpdateBet(ctx context.Context, db bun.IDB, bet *Bet) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewUpdate().
		Model(bet).
		Column("status", "settled_payout", "settled_at").
		WherePK().
		Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.UpdateBet: %w", err)
	}

	return nil
}

func (r *Impl) ListBetsForMarket(ctx context.Context, db bun.IDB, marketID int64) ([]Bet, error) {
	if db == nil {
		db = r.db
	}

	bets := make([]Bet, 0, 16)
	if err := db.NewSelect().
		Model(&bets).
		Where("market_id = ?", marketID).
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListBetsForMarket: %w", err)
	}

	return bets, nil
}

func (r *Impl) ListBetsForUserAndMarket(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, marketID int64) ([]Bet, error) {
	if db == nil {
		db = r.db
	}

	bets := make([]Bet, 0, 4)
	if err := db.NewSelect().
		Model(&bets).
		Where("club_uuid = ?", clubUUID).
		Where("user_uuid = ?", userUUID).
		Where("market_id = ?", marketID).
		OrderExpr("created_at DESC, id DESC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListBetsForUserAndMarket: %w", err)
	}

	return bets, nil
}

func (r *Impl) CreateAuditLog(ctx context.Context, db bun.IDB, log *AuditLog) error {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewInsert().Model(log).Exec(ctx); err != nil {
		return fmt.Errorf("bettingdb.CreateAuditLog: %w", err)
	}

	return nil
}

// ListOpenMarketsToLock returns all markets with status='open' whose locks_at
// is at or before the given time, across all clubs. Used by the market worker
// to lock overdue markets and emit BettingMarketLockedV1 events.
func (r *Impl) ListOpenMarketsToLock(ctx context.Context, db bun.IDB, now time.Time) ([]Market, error) {
	if db == nil {
		db = r.db
	}

	markets := make([]Market, 0)
	if err := db.NewSelect().
		Model(&markets).
		Where("status = ?", "open").
		Where("locks_at <= ?", now).
		OrderExpr("locks_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.ListOpenMarketsToLock: %w", err)
	}

	return markets, nil
}

// AcquireWalletBalance ensures a balance row exists for (club, user, season) and
// returns it under a SELECT FOR UPDATE row lock. Must be called inside a
// transaction. The lock serializes concurrent PlaceBet calls for the same user,
// eliminating the need for SERIALIZABLE isolation on the bet-placement path.
func (r *Impl) AcquireWalletBalance(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string) (*WalletBalance, error) {
	if db == nil {
		db = r.db
	}

	if _, err := db.NewRaw(`
		INSERT INTO betting_wallet_balances (club_uuid, user_uuid, season_id, balance, reserved)
		VALUES (?, ?, ?, 0, 0)
		ON CONFLICT (club_uuid, user_uuid, season_id) DO NOTHING
	`, clubUUID, userUUID, seasonID).Exec(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.AcquireWalletBalance insert: %w", err)
	}

	wb := new(WalletBalance)
	if err := db.NewSelect().
		Model(wb).
		Where("club_uuid = ?", clubUUID).
		Where("user_uuid = ?", userUUID).
		Where("season_id = ?", seasonID).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("bettingdb.AcquireWalletBalance lock: %w", err)
	}

	return wb, nil
}

// ApplyWalletBalanceDelta atomically adds balanceDelta to balance and
// reservedDelta to reserved for the given (club, user, season) row, creating
// the row if it does not yet exist. Safe to call outside a FOR UPDATE lock for
// settlement and admin-adjustment paths that run under SERIALIZABLE isolation
// or are otherwise not subject to concurrent balance-check races.
func (r *Impl) ApplyWalletBalanceDelta(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, seasonID string, balanceDelta, reservedDelta int) error {
	if db == nil {
		db = r.db
	}

	// The INSERT path uses GREATEST(…, 0) so that a row is never seeded with a
	// negative reserved value (which would violate the CHECK constraint).  The
	// UPDATE path uses the raw signed deltas so reservations are correctly
	// released (reservedDelta < 0) when a row already exists.
	// We use UPDATE-first / INSERT-fallback instead of INSERT ON CONFLICT so that
	// the CHECK constraint (balance - reserved >= 0) is evaluated against the
	// truly existing balance rather than the INSERT-proposed zeros.
	res, err := db.NewRaw(`
		UPDATE betting_wallet_balances
		SET balance  = balance  + ?,
		    reserved = reserved + ?
		WHERE club_uuid = ? AND user_uuid = ? AND season_id = ?
	`, balanceDelta, reservedDelta, clubUUID, userUUID, seasonID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("bettingdb.ApplyWalletBalanceDelta update: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("bettingdb.ApplyWalletBalanceDelta rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if _, err := db.NewRaw(`
			INSERT INTO betting_wallet_balances (club_uuid, user_uuid, season_id, balance, reserved)
			VALUES (?, ?, ?, GREATEST(?, 0), GREATEST(?, 0))
			ON CONFLICT (club_uuid, user_uuid, season_id) DO NOTHING
		`, clubUUID, userUUID, seasonID, balanceDelta, reservedDelta).Exec(ctx); err != nil {
			return fmt.Errorf("bettingdb.ApplyWalletBalanceDelta insert: %w", err)
		}
	}

	return nil
}
