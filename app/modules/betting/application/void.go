package bettingservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *BettingService) VoidRoundMarkets(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) ([]MarketVoidResult, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "VoidRoundMarkets", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.VoidRoundMarkets")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.round_id", roundID.String()),
			attribute.String("betting.guild_id", string(guildID)),
			attribute.String("betting.void_source", source),
			attribute.String("betting.void_reason", reason),
		)
	}

	run := func(ctx context.Context, db bun.IDB) ([]MarketVoidResult, error) {
		clubUUID, err := s.userRepo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve club uuid by guild id: %w", err)
		}

		markets, err := s.repo.ListMarketsByRound(ctx, db, clubUUID, roundID.UUID())
		if err != nil {
			return nil, fmt.Errorf("load betting markets by round: %w", err)
		}

		var results []MarketVoidResult
		for idx := range markets {
			if _, err := s.voidMarket(ctx, db, &markets[idx], actorUUID, reason, source); err != nil {
				return nil, err
			}
			results = append(results, MarketVoidResult{
				GuildID:  guildID,
				ClubUUID: clubUUID.String(),
				RoundID:  roundID,
				MarketID: markets[idx].ID,
				Reason:   blankIfEmpty(reason, markets[idx].VoidReason),
			})
		}
		return results, nil
	}

	results, err := runInTx(ctx, s.db, &sql.TxOptions{Isolation: sql.LevelSerializable}, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "VoidRoundMarkets", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.operation.failed", "VoidRoundMarkets failed", err,
			attr.String("guild_id", string(guildID)),
			attr.String("void_source", source),
		)
		return nil, err
	}

	s.metrics.RecordRoundVoided(ctx, source)
	s.metrics.RecordOperationSuccess(ctx, "VoidRoundMarkets", "betting")
	s.metrics.RecordOperationDuration(ctx, "VoidRoundMarkets", "betting", time.Since(start))
	s.metrics.RecordSettlementDuration(ctx, source, time.Since(start))

	s.logInfo(ctx, "betting.round.voided", "round markets voided",
		attr.String("guild_id", string(guildID)),
		attr.String("void_source", source),
		attr.String("void_reason", reason),
		attr.Int("market_count", len(results)),
	)

	return results, nil
}

func (s *BettingService) voidMarket(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	actorUUID *uuid.UUID,
	reason string,
	source string,
) (bool, error) {
	bets, err := s.repo.ListBetsForMarket(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market bets: %w", err)
	}
	changed, err := s.applyVoidSettlement(ctx, db, market, bets, actorUUID, reason, source, false)
	if err != nil {
		return false, err
	}
	if changed {
		s.metrics.RecordMarketVoided(ctx, market.MarketType, reason)
		s.logInfo(ctx, "betting.market.voided", "market voided",
			attr.String("market_type", market.MarketType),
			attr.String("void_reason", reason),
		)
	}
	return changed, nil
}

func (s *BettingService) applyVoidSettlement(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	bets []bettingdb.Bet,
	actorUUID *uuid.UUID,
	reason string,
	source string,
	// createEntryForAccepted controls whether a refund journal entry is written
	// for bets still in the accepted (reserved) state. Deletion-induced voids
	// pass false (stake never debited). Settlement-induced voids pass true so an
	// explicit refund record is always written.
	createEntryForAccepted bool,
) (bool, error) {
	now := time.Now().UTC()
	changed := false
	for idx := range bets {
		targetPayout := bets[idx].Stake
		if !betNeedsUpdate(&bets[idx], voidedBetStatus, targetPayout) {
			continue
		}
		// For accepted bets, the stake is tracked only in reserved (via the accepted
		// bet itself) — GetWalletJournalBalance excludes stake_reserved entries, so
		// nothing was ever debited from bettingBalance. Releasing reserved (by setting
		// status to voided) is sufficient; no journal entry is needed.
		// A journal entry is only required when correcting a previously settled bet
		// (where a payout was already credited or the stake was previously debited).
		delta := targetPayout - bets[idx].SettledPayout
		if delta != 0 && (bets[idx].Status != acceptedBetStatus || createEntryForAccepted) {
			entryType := marketRefundEntry
			if delta < 0 {
				entryType = marketCorrectionEntry
			}
			if err := s.repo.CreateWalletJournalEntry(ctx, db, &bettingdb.WalletJournalEntry{
				ClubUUID:  bets[idx].ClubUUID,
				UserUUID:  bets[idx].UserUUID,
				SeasonID:  bets[idx].SeasonID,
				EntryType: entryType,
				Amount:    delta,
				Reason:    fmt.Sprintf("Market voided: %s", reason),
				CreatedBy: createdByValue(actorUUID, source),
			}); err != nil {
				return false, fmt.Errorf("create void journal entry: %w", err)
			}
		}

		// Keep the wallet balance projection in sync.
		// For accepted bets: reserved is released; balance is credited only if a
		// refund journal entry was written (createEntryForAccepted=true).
		// For previously settled bets: apply the correction delta to balance;
		// reserved is already zero (bet was no longer in accepted state).
		var balanceDelta, reservedDelta int
		if bets[idx].Status == acceptedBetStatus {
			reservedDelta = -bets[idx].Stake
			if createEntryForAccepted {
				balanceDelta = bets[idx].Stake
			}
		} else if delta != 0 {
			balanceDelta = delta
		}
		if balanceDelta != 0 || reservedDelta != 0 {
			if err := s.repo.ApplyWalletBalanceDelta(ctx, db, bets[idx].ClubUUID, bets[idx].UserUUID, bets[idx].SeasonID, balanceDelta, reservedDelta); err != nil {
				return false, fmt.Errorf("update wallet balance on void: %w", err)
			}
		}

		bets[idx].Status = voidedBetStatus
		bets[idx].SettledPayout = targetPayout
		bets[idx].SettledAt = &now
		if err := s.repo.UpdateBet(ctx, db, &bets[idx]); err != nil {
			return false, fmt.Errorf("update voided bet: %w", err)
		}
		changed = true
	}

	if !marketNeedsUpdate(market, voidedMarketStatus, "", reason, reason, source) && !changed {
		return false, nil
	}

	market.Status = voidedMarketStatus
	market.ResolvedOptionKey = ""
	market.VoidReason = reason
	market.ResultSummary = reason
	market.LastResultSource = source
	market.SettledAt = &now
	market.SettlementVersion++
	market.UpdatedAt = now
	if err := s.repo.UpdateMarket(ctx, db, market); err != nil {
		if errors.Is(err, bettingdb.ErrSettlementVersionConflict) {
			return false, nil
		}
		return false, fmt.Errorf("update voided market: %w", err)
	}

	if err := s.repo.CreateAuditLog(ctx, db, &bettingdb.AuditLog{
		ClubUUID:      market.ClubUUID,
		MarketID:      int64Ptr(market.ID),
		RoundID:       uuidPtr(market.RoundID),
		ActorUserUUID: actorUUID,
		Action:        "market_voided",
		Reason:        reason,
		Metadata:      fmt.Sprintf("source=%s", source),
	}); err != nil {
		return false, fmt.Errorf("create void audit log: %w", err)
	}

	return true, nil
}
