package bettingservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// EnsureMarketsForGuild resolves the club UUID for the given guild, checks
// that betting entitlements are active, then generates all eligible markets for
// every upcoming round. It returns a result slice that the caller (typically
// the background worker) can use to emit domain events.
func (s *BettingService) EnsureMarketsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]MarketGeneratedResult, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "EnsureMarketsForGuild", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.EnsureMarketsForGuild")
		defer span.End()
		span.SetAttributes(attribute.String("betting.guild_id", string(guildID)))
	}

	run := func(ctx context.Context, db bun.IDB) ([]MarketGeneratedResult, error) {
		clubUUID, err := s.userRepo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve club uuid by guild id: %w", err)
		}

		entitlements, err := s.guildRepo.ResolveEntitlements(ctx, db, guildID)
		if err != nil {
			return nil, fmt.Errorf("resolve entitlements for guild %s: %w", guildID, err)
		}
		bettingAccess := entitlements.Features[guildtypes.ClubFeatureBetting]
		if bettingAccess.State == guildtypes.FeatureAccessStateDisabled ||
			bettingAccess.State == guildtypes.FeatureAccessStateFrozen {
			return nil, nil
		}

		seasonID := defaultSeasonID
		if activeSeason, err := s.leaderboardRepo.GetActiveSeason(ctx, db, string(guildID)); err == nil && activeSeason != nil {
			seasonID = activeSeason.ID
		}

		upcoming, err := s.roundRepo.GetAllUpcomingRoundsInWindow(ctx, db, 48*time.Hour)
		if err != nil {
			return nil, fmt.Errorf("fetch upcoming rounds for guild %s: %w", guildID, err)
		}

		var (
			results      []MarketGeneratedResult
			eligible     int
			failedRounds int
		)
		for _, round := range upcoming {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}

			if round.GuildID != guildID {
				continue
			}

			participants, _, err := s.collectEligibleParticipants(ctx, db, clubUUID, round)
			if err != nil || len(participants) < 2 {
				continue
			}
			eligible++

			roundResults, err := s.ensureAllMarketsForRound(ctx, db, clubUUID, seasonID, guildID, round, participants)
			if err != nil {
				s.logWarn(ctx, "betting.market.ensure_error", "ensure markets error",
					attr.String("guild_id", string(guildID)),
					attr.Any("round_id", round.ID),
					attr.Error(err),
				)
				failedRounds++
				continue
			}
			results = append(results, roundResults...)
		}
		if eligible > 0 && failedRounds == eligible {
			return nil, fmt.Errorf("all %d eligible rounds failed market generation for guild %s", failedRounds, guildID)
		}
		return results, nil
	}

	results, err := runInTx(ctx, s.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted}, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "EnsureMarketsForGuild", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.operation.failed", "EnsureMarketsForGuild failed", err,
			attr.String("guild_id", string(guildID)),
		)
		return nil, err
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("betting.market_count", len(results)))
	}

	s.metrics.RecordOperationSuccess(ctx, "EnsureMarketsForGuild", "betting")
	s.metrics.RecordOperationDuration(ctx, "EnsureMarketsForGuild", "betting", time.Since(start))

	if len(results) > 0 {
		s.logInfo(ctx, "betting.market.ensure_complete", "markets ensured for guild",
			attr.String("guild_id", string(guildID)),
			attr.Int("market_count", len(results)),
		)
	}

	return results, nil
}

// ensureAllMarketsForRound generates all eligible market types for a single round.
func (s *BettingService) ensureAllMarketsForRound(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) ([]MarketGeneratedResult, error) {
	fieldSize := len(participants)
	var results []MarketGeneratedResult

	s.metrics.RecordEligibleParticipantCount(ctx, winnerMarketType, fieldSize)

	// Winner market — always generated when field >= 2.
	if market, _, err := s.ensureWinnerMarket(ctx, db, clubUUID, seasonID, guildID, round, participants); err != nil {
		return nil, err
	} else if market != nil {
		results = append(results, MarketGeneratedResult{
			GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: round.ID,
			MarketID: market.ID, MarketType: winnerMarketType,
		})
	}

	// 2nd place — requires 3+ players.
	if fieldSize >= 3 {
		if market, _, err := s.ensurePlacement2ndMarket(ctx, db, clubUUID, seasonID, guildID, round, participants); err != nil {
			return nil, err
		} else if market != nil {
			results = append(results, MarketGeneratedResult{
				GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: round.ID,
				MarketID: market.ID, MarketType: placement2ndMarketType,
			})
		}
	}

	// 3rd place — requires 4+ players.
	if fieldSize >= 4 {
		if market, _, err := s.ensurePlacement3rdMarket(ctx, db, clubUUID, seasonID, guildID, round, participants); err != nil {
			return nil, err
		} else if market != nil {
			results = append(results, MarketGeneratedResult{
				GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: round.ID,
				MarketID: market.ID, MarketType: placement3rdMarketType,
			})
		}
	}

	// Last place — requires 3+ players (with 2, last = 2nd which is covered by winner market).
	if fieldSize >= 3 {
		if market, _, err := s.ensurePlacementLastMarket(ctx, db, clubUUID, seasonID, guildID, round, participants); err != nil {
			return nil, err
		} else if market != nil {
			results = append(results, MarketGeneratedResult{
				GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: round.ID,
				MarketID: market.ID, MarketType: placementLastMarketType,
			})
		}
	}

	// Over/Under — generated whenever field >= 2.
	if market, _, err := s.ensureOverUnderMarket(ctx, db, clubUUID, seasonID, guildID, round, participants); err != nil {
		return nil, err
	} else if market != nil {
		results = append(results, MarketGeneratedResult{
			GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: round.ID,
			MarketID: market.ID, MarketType: overUnderMarketType,
		})
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// getOrBuild helpers (used by both ensure functions and query layer)
// ---------------------------------------------------------------------------

func (s *BettingService) getOrBuildWinnerMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, bool, error) {
	return s.getOrBuildMarket(ctx, db, clubUUID, seasonID, guildID, round, winnerMarketType, winnerMarketTitle(round), func() ([]pricedOption, error) {
		return s.oddsEngine.priceWinnerOptions(ctx, db, guildID, participants)
	})
}

func (s *BettingService) getOrBuildPlacement2ndMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, bool, error) {
	return s.getOrBuildMarket(ctx, db, clubUUID, seasonID, guildID, round, placement2ndMarketType, placement2ndMarketTitle(round), func() ([]pricedOption, error) {
		return s.oddsEngine.pricePlacementOptions(ctx, db, guildID, participants, 2)
	})
}

func (s *BettingService) getOrBuildPlacement3rdMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, bool, error) {
	return s.getOrBuildMarket(ctx, db, clubUUID, seasonID, guildID, round, placement3rdMarketType, placement3rdMarketTitle(round), func() ([]pricedOption, error) {
		return s.oddsEngine.pricePlacementOptions(ctx, db, guildID, participants, 3)
	})
}

func (s *BettingService) getOrBuildPlacementLastMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, bool, error) {
	lastPosition := len(participants)
	return s.getOrBuildMarket(ctx, db, clubUUID, seasonID, guildID, round, placementLastMarketType, placementLastMarketTitle(round), func() ([]pricedOption, error) {
		return s.oddsEngine.pricePlacementOptions(ctx, db, guildID, participants, lastPosition)
	})
}

func (s *BettingService) getOrBuildOverUnderMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, bool, error) {
	return s.getOrBuildMarket(ctx, db, clubUUID, seasonID, guildID, round, overUnderMarketType, overUnderMarketTitle(round), func() ([]pricedOption, error) {
		return s.oddsEngine.priceOverUnderOptions(ctx, db, guildID, participants)
	})
}

// getOrBuildMarket is the shared read-or-price logic for all market types.
func (s *BettingService) getOrBuildMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	marketType string,
	title string,
	priceFn func() ([]pricedOption, error),
) (*bettingdb.Market, []pricedOption, bool, error) {
	market, err := s.repo.GetMarketByRound(ctx, db, clubUUID, seasonID, round.ID.UUID(), marketType)
	if err != nil {
		return nil, nil, false, fmt.Errorf("load betting market: %w", err)
	}
	if market != nil {
		options, err := s.repo.ListMarketOptions(ctx, db, market.ID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("load betting market options: %w", err)
		}
		return market, toPricedOptions(options), false, nil
	}

	options, err := priceFn()
	if err != nil {
		return nil, nil, false, err
	}

	return &bettingdb.Market{
		ClubUUID:   clubUUID,
		SeasonID:   seasonID,
		RoundID:    round.ID.UUID(),
		MarketType: marketType,
		Title:      title,
		Status:     openMarketStatus,
		LocksAt:    roundStartTime(round),
	}, options, true, nil
}

// ---------------------------------------------------------------------------
// ensure* functions (persist + TOCTOU handling)
// ---------------------------------------------------------------------------

func (s *BettingService) ensureWinnerMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, error) {
	return s.ensureMarket(ctx, db, clubUUID, seasonID, guildID, round, winnerMarketType,
		func() (*bettingdb.Market, []pricedOption, bool, error) {
			return s.getOrBuildWinnerMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		})
}

func (s *BettingService) ensurePlacement2ndMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, error) {
	return s.ensureMarket(ctx, db, clubUUID, seasonID, guildID, round, placement2ndMarketType,
		func() (*bettingdb.Market, []pricedOption, bool, error) {
			return s.getOrBuildPlacement2ndMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		})
}

func (s *BettingService) ensurePlacement3rdMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, error) {
	return s.ensureMarket(ctx, db, clubUUID, seasonID, guildID, round, placement3rdMarketType,
		func() (*bettingdb.Market, []pricedOption, bool, error) {
			return s.getOrBuildPlacement3rdMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		})
}

func (s *BettingService) ensurePlacementLastMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, error) {
	return s.ensureMarket(ctx, db, clubUUID, seasonID, guildID, round, placementLastMarketType,
		func() (*bettingdb.Market, []pricedOption, bool, error) {
			return s.getOrBuildPlacementLastMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		})
}

func (s *BettingService) ensureOverUnderMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
) (*bettingdb.Market, []pricedOption, error) {
	return s.ensureMarket(ctx, db, clubUUID, seasonID, guildID, round, overUnderMarketType,
		func() (*bettingdb.Market, []pricedOption, bool, error) {
			return s.getOrBuildOverUnderMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		})
}

// ensureMarket is the shared persist + TOCTOU race handler for all market types.
func (s *BettingService) ensureMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	marketType string,
	buildFn func() (*bettingdb.Market, []pricedOption, bool, error),
) (*bettingdb.Market, []pricedOption, error) {
	_ = guildID // reserved for future use
	market, options, ephemeral, err := buildFn()
	if err != nil {
		return nil, nil, err
	}
	if !ephemeral {
		return market, options, nil
	}

	if err := s.repo.CreateMarket(ctx, db, market); err != nil {
		// TOCTOU: another concurrent worker may have inserted the same market.
		if existing, getErr := s.repo.GetMarketByRound(ctx, db, clubUUID, seasonID, round.ID.UUID(), marketType); getErr == nil && existing != nil {
			existingOptions, optErr := s.repo.ListMarketOptions(ctx, db, existing.ID)
			if optErr != nil {
				return nil, nil, fmt.Errorf("load existing market options after conflict: %w", optErr)
			}
			return existing, toPricedOptions(existingOptions), nil
		}
		return nil, nil, fmt.Errorf("create betting market: %w", err)
	}

	dbOptions := make([]bettingdb.MarketOption, 0, len(options))
	for idx, option := range options {
		dbOptions = append(dbOptions, bettingdb.MarketOption{
			MarketID:            market.ID,
			OptionKey:           option.optionKey,
			ParticipantMemberID: string(option.memberID),
			Label:               option.label,
			ProbabilityBps:      option.probabilityBps,
			DecimalOddsCents:    option.decimalOddsCents,
			DisplayOrder:        idx,
			Metadata:            option.metadata,
		})
	}
	if err := s.repo.CreateMarketOptions(ctx, db, dbOptions); err != nil {
		return nil, nil, fmt.Errorf("create betting market options: %w", err)
	}

	s.metrics.RecordMarketCreated(ctx, marketType)
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.AddEvent("market.created",
			trace.WithAttributes(
				attribute.String("betting.market_type", marketType),
				attribute.String("betting.round_id", round.ID.String()),
				attribute.Int("betting.option_count", len(dbOptions)),
			),
		)
	}
	s.logInfo(ctx, "betting.market.created", "market created",
		attr.String("market_type", marketType),
		attr.Any("round_id", round.ID),
		attr.Int("option_count", len(dbOptions)),
	)

	return market, options, nil
}

// LockDueMarkets finds all open markets whose locks_at has passed, updates
// their status to locked within a single transaction, and returns results for
// the caller to emit BettingMarketLockedV1 events.
func (s *BettingService) LockDueMarkets(ctx context.Context) ([]MarketLockResult, error) {
	now := time.Now().UTC()
	s.metrics.RecordOperationAttempt(ctx, "LockDueMarkets", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.LockDueMarkets")
		defer span.End()
	}

	run := func(ctx context.Context, db bun.IDB) ([]MarketLockResult, error) {
		markets, err := s.repo.ListOpenMarketsToLock(ctx, db, now)
		if err != nil {
			return nil, fmt.Errorf("list open markets to lock: %w", err)
		}
		if len(markets) == 0 {
			return nil, nil
		}

		results := make([]MarketLockResult, 0, len(markets))
		for i := range markets {
			m := &markets[i]
			m.Status = lockedMarketStatus
			if err := s.repo.UpdateMarket(ctx, db, m); err != nil {
				return nil, fmt.Errorf("lock market %d: %w", m.ID, err)
			}

			guildID, err := s.userRepo.GetDiscordGuildIDByClubUUID(ctx, db, m.ClubUUID)
			if err != nil {
				s.logWarn(ctx, "betting.market.lock.guild_resolve_failed", "could not resolve guild for locked market",
					attr.Int64("market_id", m.ID),
				)
				continue
			}

			results = append(results, MarketLockResult{
				GuildID:  guildID,
				ClubUUID: m.ClubUUID.String(),
				RoundID:  sharedtypes.RoundID(m.RoundID),
				MarketID: m.ID,
			})
			s.logInfo(ctx, "betting.market.locked", "market locked",
				attr.Int64("market_id", m.ID),
				attr.String("club_uuid", m.ClubUUID.String()),
			)
		}
		return results, nil
	}

	results, err := runInTx(ctx, s.db, nil, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "LockDueMarkets", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.operation.failed", "LockDueMarkets failed", err)
		return nil, err
	}

	s.metrics.RecordOperationSuccess(ctx, "LockDueMarkets", "betting")
	return results, nil
}
