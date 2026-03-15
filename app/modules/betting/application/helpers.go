package bettingservice

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// runInTx runs fn inside a serializable transaction. If db is nil (test mode)
// the function is called directly without a transaction.
func runInTx[T any](
	ctx context.Context,
	db *bun.DB,
	opts *sql.TxOptions,
	run func(context.Context, bun.IDB) (T, error),
) (T, error) {
	var zero T
	if db == nil {
		return run(ctx, nil)
	}

	var result T
	err := db.RunInTx(ctx, opts, func(ctx context.Context, tx bun.Tx) error {
		var txErr error
		result, txErr = run(ctx, tx)
		return txErr
	})
	if err != nil {
		return zero, err
	}
	return result, nil
}

// mirrorTx runs fn inside a transaction. If db is nil (test mode) the function
// is called directly without a transaction. Unlike runInTx, fn returns only
// error (no result value), which is convenient for fire-and-forget journal writes.
func mirrorTx(
	ctx context.Context,
	db *bun.DB,
	fn func(context.Context, bun.IDB) error,
) error {
	if db == nil {
		return fn(ctx, nil)
	}
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, tx)
	})
}

// ---------------------------------------------------------------------------
// Option / market conversion helpers
// ---------------------------------------------------------------------------

func toAPIOptions(options []pricedOption) []BettingMarketOption {
	apiOptions := make([]BettingMarketOption, 0, len(options))
	for _, option := range options {
		apiOptions = append(apiOptions, BettingMarketOption{
			OptionKey:          option.optionKey,
			MemberID:           string(option.memberID),
			Label:              option.label,
			ProbabilityPercent: int(math.Round(float64(option.probabilityBps) / 100)),
			DecimalOdds:        decimalOddsFromCents(option.decimalOddsCents),
			Metadata:           option.metadata,
		})
	}
	return apiOptions
}

func toPricedOptions(options []bettingdb.MarketOption) []pricedOption {
	priced := make([]pricedOption, 0, len(options))
	for _, option := range options {
		priced = append(priced, pricedOption{
			optionKey:        option.OptionKey,
			memberID:         sharedtypes.DiscordID(option.ParticipantMemberID),
			label:            option.Label,
			probabilityBps:   option.ProbabilityBps,
			decimalOddsCents: option.DecimalOddsCents,
			metadata:         option.Metadata,
		})
	}
	return priced
}

func findOptionByKey(options []pricedOption, key string) (pricedOption, bool) {
	for _, option := range options {
		if option.optionKey == key {
			return option, true
		}
	}
	return pricedOption{}, false
}

// ---------------------------------------------------------------------------
// Round / market value helpers
// ---------------------------------------------------------------------------

func winnerMarketTitle(round *roundtypes.Round) string {
	return fmt.Sprintf("%s winner", round.Title.String())
}

func placement2ndMarketTitle(round *roundtypes.Round) string {
	return fmt.Sprintf("%s 2nd place", round.Title.String())
}

func placement3rdMarketTitle(round *roundtypes.Round) string {
	return fmt.Sprintf("%s 3rd place", round.Title.String())
}

func placementLastMarketTitle(round *roundtypes.Round) string {
	return fmt.Sprintf("%s last place", round.Title.String())
}

func overUnderMarketTitle(round *roundtypes.Round) string {
	return fmt.Sprintf("%s score over/under", round.Title.String())
}

// marketTypeProhibitsSelfBet returns true for market types where a player must
// not bet on themselves. The winner market is excluded intentionally.
func marketTypeProhibitsSelfBet(marketType string) bool {
	switch marketType {
	case placement2ndMarketType, placement3rdMarketType,
		placementLastMarketType, overUnderMarketType:
		return true
	default:
		return false
	}
}

func roundStartTime(round *roundtypes.Round) time.Time {
	if round == nil || round.StartTime == nil {
		return time.Time{}
	}
	return round.StartTime.AsTime().UTC()
}

func effectiveMarketStatus(status string, locksAt time.Time) string {
	if strings.EqualFold(status, openMarketStatus) && !locksAt.IsZero() && !time.Now().UTC().Before(locksAt) {
		return lockedMarketStatus
	}
	if status == "" {
		return openMarketStatus
	}
	return status
}

func marketStatusValue(market *bettingdb.Market) string {
	if market == nil {
		return openMarketStatus
	}
	return market.Status
}

func marketResultValue(market *bettingdb.Market) string {
	if market == nil {
		return ""
	}
	return market.ResultSummary
}

func marketIDValue(market *bettingdb.Market) int64 {
	if market == nil {
		return 0
	}
	return market.ID
}

// ---------------------------------------------------------------------------
// Bet calculation helpers
// ---------------------------------------------------------------------------

func calculatePotentialPayout(stake, decimalOddsCents int) int {
	product := int64(stake) * int64(decimalOddsCents)
	return int(math.Round(float64(product) / 100))
}

func decimalOddsFromCents(cents int) float64 {
	return float64(cents) / 100
}

func toTicket(bet bettingdb.Bet) BetTicket {
	return BetTicket{
		ID:              bet.ID,
		RoundID:         bet.RoundID.String(),
		MarketType:      bet.MarketType,
		SelectionKey:    bet.SelectionKey,
		SelectionLabel:  bet.SelectionLabel,
		Stake:           bet.Stake,
		DecimalOdds:     decimalOddsFromCents(bet.DecimalOddsCents),
		PotentialPayout: bet.PotentialPayout,
		SettledPayout:   bet.SettledPayout,
		Status:          bet.Status,
		SettledAt:       bet.SettledAt,
		CreatedAt:       bet.CreatedAt,
	}
}

func betNeedsUpdate(bet *bettingdb.Bet, status string, payout int) bool {
	if bet == nil {
		return false
	}
	return bet.Status != status || bet.SettledPayout != payout || bet.SettledAt == nil
}

func marketNeedsUpdate(market *bettingdb.Market, status, resolvedKey, voidReason, summary, source string) bool {
	if market == nil {
		return false
	}
	return market.Status != status ||
		market.ResolvedOptionKey != resolvedKey ||
		market.VoidReason != voidReason ||
		market.ResultSummary != summary ||
		market.LastResultSource != source ||
		market.SettledAt == nil
}

// ---------------------------------------------------------------------------
// Misc helpers
// ---------------------------------------------------------------------------

func createdByValue(actorUUID *uuid.UUID, source string) string {
	if actorUUID != nil && *actorUUID != uuid.Nil {
		return actorUUID.String()
	}
	return source
}

func blankIfEmpty(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func int64Ptr(v int64) *int64 {
	return &v
}

func uuidPtr(v uuid.UUID) *uuid.UUID {
	return &v
}
