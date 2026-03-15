package bettingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *BettingService) GetOverview(ctx context.Context, clubUUID, userUUID uuid.UUID) (*Overview, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "GetOverview", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.GetOverview")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.club_uuid", clubUUID.String()),
		)
	}

	guildID, access, err := s.resolveAccess(ctx, nil, clubUUID, userUUID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetOverview", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}
	if access.State == guildtypes.FeatureAccessStateDisabled {
		s.metrics.RecordAccessDenied(ctx, "disabled")
		s.metrics.RecordOperationFailure(ctx, "GetOverview", "betting")
		s.logWarn(ctx, "betting.overview.access.denied", "GetOverview access denied: feature disabled",
			attr.UUIDValue("club_uuid", clubUUID),
		)
		return nil, ErrFeatureDisabled
	}

	wallet, err := s.resolveWallet(ctx, nil, clubUUID, userUUID, guildID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetOverview", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	settings, err := s.repo.GetMemberSettings(ctx, nil, clubUUID, userUUID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetOverview", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("load betting settings: %w", err)
	}

	ledgerEntries, err := s.repo.ListWalletJournal(ctx, nil, clubUUID, userUUID, wallet.seasonID, walletHistorySize)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetOverview", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("load betting wallet history: %w", err)
	}

	overview := &Overview{
		ClubUUID:     clubUUID.String(),
		GuildID:      string(guildID),
		SeasonID:     wallet.seasonID,
		SeasonName:   wallet.seasonName,
		AccessState:  string(access.State),
		AccessSource: string(access.Source),
		ReadOnly:     access.State != guildtypes.FeatureAccessStateEnabled,
		Wallet: WalletSnapshot{
			SeasonPoints:      wallet.seasonPoints,
			AdjustmentBalance: wallet.bettingBalance,
			Available:         wallet.bettingBalance - wallet.reserved,
			Reserved:          wallet.reserved,
		},
		Settings: MemberSettings{},
		Journal:  make([]WalletJournal, 0, len(ledgerEntries)),
	}

	if settings != nil {
		overview.Settings.OptOutTargeting = settings.OptOutTargeting
		overview.Settings.UpdatedAt = settings.UpdatedAt
	}

	for _, entry := range ledgerEntries {
		overview.Journal = append(overview.Journal, WalletJournal{
			ID:        entry.ID,
			EntryType: entry.EntryType,
			Amount:    entry.Amount,
			Reason:    entry.Reason,
			CreatedAt: entry.CreatedAt,
		})
	}

	s.metrics.RecordOperationSuccess(ctx, "GetOverview", "betting")
	s.metrics.RecordOperationDuration(ctx, "GetOverview", "betting", time.Since(start))

	return overview, nil
}

func (s *BettingService) GetNextRoundMarket(ctx context.Context, clubUUID, userUUID uuid.UUID) (*NextRoundMarket, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "GetNextRoundMarket", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.GetNextRoundMarket")
		defer span.End()
		span.SetAttributes(attribute.String("betting.club_uuid", clubUUID.String()))
	}

	guildID, access, err := s.resolveAccess(ctx, nil, clubUUID, userUUID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}
	if access.State == guildtypes.FeatureAccessStateDisabled {
		s.metrics.RecordAccessDenied(ctx, "disabled")
		s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
		s.logWarn(ctx, "betting.next_round_market.access.denied", "GetNextRoundMarket access denied: feature disabled",
			attr.UUIDValue("club_uuid", clubUUID),
		)
		return nil, ErrFeatureDisabled
	}

	wallet, err := s.resolveWallet(ctx, nil, clubUUID, userUUID, guildID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	round, participants, warnings, err := s.findNextEligibleRound(ctx, nil, clubUUID, guildID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	// Resolve bettor's Discord ID for self-bet restriction tagging.
	bettorDiscordID := ""
	if bettor, uErr := s.userRepo.GetUserByUUID(ctx, nil, userUUID); uErr == nil && bettor != nil {
		bettorDiscordID = string(bettor.GetUserID())
	}

	allMarkets, err := s.buildAllMarketViews(ctx, nil, clubUUID, wallet.seasonID, guildID, round, participants, bettorDiscordID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	// Fetch user bets across all markets that exist in the DB.
	userBets := make([]BetTicket, 0)
	for _, bm := range allMarkets {
		if bm.ID == 0 {
			continue
		}
		bets, err := s.repo.ListBetsForUserAndMarket(ctx, nil, clubUUID, userUUID, bm.ID)
		if err != nil {
			s.metrics.RecordOperationFailure(ctx, "GetNextRoundMarket", "betting")
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.RecordError(err)
			}
			return nil, fmt.Errorf("load user market bets: %w", err)
		}
		for _, bet := range bets {
			userBets = append(userBets, toTicket(bet))
		}
	}

	s.metrics.RecordOperationSuccess(ctx, "GetNextRoundMarket", "betting")
	s.metrics.RecordOperationDuration(ctx, "GetNextRoundMarket", "betting", time.Since(start))

	apiMarkets := make([]BettingMarket, 0, len(allMarkets))
	for _, bm := range allMarkets {
		apiMarkets = append(apiMarkets, bm.BettingMarket)
	}

	var firstMarket *BettingMarket
	if len(apiMarkets) > 0 {
		m := apiMarkets[0]
		firstMarket = &m
	}

	return &NextRoundMarket{
		ClubUUID:    clubUUID.String(),
		GuildID:     string(guildID),
		SeasonID:    wallet.seasonID,
		AccessState: string(access.State),
		ReadOnly:    access.State != guildtypes.FeatureAccessStateEnabled,
		Wallet: WalletSnapshot{
			SeasonPoints:      wallet.seasonPoints,
			AdjustmentBalance: wallet.bettingBalance,
			Available:         wallet.bettingBalance - wallet.reserved,
			Reserved:          wallet.reserved,
		},
		Round: BettingRound{
			ID:        round.ID.String(),
			Title:     round.Title.String(),
			StartTime: roundStartTime(round),
		},
		Market:   firstMarket,
		Markets:  apiMarkets,
		UserBets: userBets,
		Warnings: warnings,
	}, nil
}

func (s *BettingService) GetAdminMarkets(ctx context.Context, clubUUID, userUUID uuid.UUID) (*AdminMarketBoard, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "GetAdminMarkets", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.GetAdminMarkets")
		defer span.End()
		span.SetAttributes(attribute.String("betting.club_uuid", clubUUID.String()))
	}

	guildID, _, err := s.resolveAdminAccess(ctx, nil, clubUUID, userUUID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetAdminMarkets", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	markets, err := s.repo.ListMarketsForClub(ctx, nil, clubUUID, adminMarketListSize)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetAdminMarkets", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("load betting admin markets: %w", err)
	}

	items := make([]AdminMarketSummary, 0, len(markets))
	for idx := range markets {
		summary, err := s.buildAdminMarketSummary(ctx, nil, guildID, &markets[idx])
		if err != nil {
			s.metrics.RecordOperationFailure(ctx, "GetAdminMarkets", "betting")
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.RecordError(err)
			}
			return nil, err
		}
		items = append(items, summary)
	}

	s.metrics.RecordOperationSuccess(ctx, "GetAdminMarkets", "betting")
	s.metrics.RecordOperationDuration(ctx, "GetAdminMarkets", "betting", time.Since(start))

	s.logInfo(ctx, "betting.admin.markets_fetched", "admin markets fetched",
		attr.UUIDValue("club_uuid", clubUUID),
		attr.Int("market_count", len(items)),
	)

	return &AdminMarketBoard{
		ClubUUID: clubUUID.String(),
		GuildID:  string(guildID),
		Markets:  items,
	}, nil
}

func (s *BettingService) buildAdminMarketSummary(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	market *bettingdb.Market,
) (AdminMarketSummary, error) {
	bets, err := s.repo.ListBetsForMarket(ctx, db, market.ID)
	if err != nil {
		return AdminMarketSummary{}, fmt.Errorf("load market bets: %w", err)
	}

	roundTitle := market.RoundID.String()
	if round, err := s.roundRepo.GetRound(ctx, db, guildID, sharedtypes.RoundID(market.RoundID)); err == nil && round != nil {
		roundTitle = round.Title.String()
	}

	summary := AdminMarketSummary{
		ID:                market.ID,
		RoundID:           market.RoundID.String(),
		RoundTitle:        roundTitle,
		MarketType:        market.MarketType,
		Title:             market.Title,
		Status:            effectiveMarketStatus(market.Status, market.LocksAt),
		LocksAt:           market.LocksAt,
		SettledAt:         market.SettledAt,
		ResultSummary:     market.ResultSummary,
		SettlementVersion: market.SettlementVersion,
		TicketCount:       len(bets),
	}

	for _, bet := range bets {
		summary.Exposure += bet.Stake
		switch bet.Status {
		case acceptedBetStatus:
			summary.AcceptedTickets++
		case wonBetStatus:
			summary.WonTickets++
		case lostBetStatus:
			summary.LostTickets++
		case voidedBetStatus:
			summary.VoidedTickets++
		}
	}

	return summary, nil
}

// GetMarketSnapshot returns the public (non-user-specific) view of the next
// upcoming betting market for a club. No wallet or per-user bet data is
// included. Used by the NATS snapshot request/reply handler.
func (s *BettingService) GetMarketSnapshot(ctx context.Context, clubUUID uuid.UUID) (*MarketSnapshot, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "GetMarketSnapshot", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.GetMarketSnapshot")
		defer span.End()
		span.SetAttributes(attribute.String("betting.club_uuid", clubUUID.String()))
	}

	guildID, access, err := s.resolveAccessByClub(ctx, nil, clubUUID)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetMarketSnapshot", "betting")
		return nil, err
	}

	snapshot := &MarketSnapshot{
		ClubUUID:    clubUUID.String(),
		GuildID:     string(guildID),
		AccessState: string(access.State),
	}

	if access.State == guildtypes.FeatureAccessStateDisabled {
		s.metrics.RecordOperationSuccess(ctx, "GetMarketSnapshot", "betting")
		s.metrics.RecordOperationDuration(ctx, "GetMarketSnapshot", "betting", time.Since(start))
		return snapshot, nil
	}

	season, err := s.leaderboardRepo.GetActiveSeason(ctx, nil, string(guildID))
	if err != nil || season == nil {
		// No active season — return snapshot without market.
		s.metrics.RecordOperationSuccess(ctx, "GetMarketSnapshot", "betting")
		s.metrics.RecordOperationDuration(ctx, "GetMarketSnapshot", "betting", time.Since(start))
		return snapshot, nil
	}
	snapshot.SeasonID = season.ID

	round, participants, _, err := s.findNextEligibleRound(ctx, nil, clubUUID, guildID)
	if err != nil || round == nil {
		s.metrics.RecordOperationSuccess(ctx, "GetMarketSnapshot", "betting")
		s.metrics.RecordOperationDuration(ctx, "GetMarketSnapshot", "betting", time.Since(start))
		return snapshot, nil
	}

	allMarkets, err := s.buildAllMarketViews(ctx, nil, clubUUID, season.ID, guildID, round, participants, "")
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "GetMarketSnapshot", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return nil, err
	}

	snapshot.Round = &BettingRound{
		ID:        round.ID.String(),
		Title:     round.Title.String(),
		StartTime: roundStartTime(round),
	}

	snapshot.Markets = make([]BettingMarket, 0, len(allMarkets))
	for _, bm := range allMarkets {
		snapshot.Markets = append(snapshot.Markets, bm.BettingMarket)
	}
	if len(snapshot.Markets) > 0 {
		m := snapshot.Markets[0]
		snapshot.Market = &m
	}

	s.metrics.RecordOperationSuccess(ctx, "GetMarketSnapshot", "betting")
	s.metrics.RecordOperationDuration(ctx, "GetMarketSnapshot", "betting", time.Since(start))
	return snapshot, nil
}

// marketViewWithID carries both the API view and the DB ID (needed for bet fetching).
type marketViewWithID struct {
	BettingMarket
	ID int64
}

// buildAllMarketViews fetches or prices all eligible markets for a round and
// returns them as API-ready BettingMarket values. bettorDiscordID is used to
// tag self_bet_restricted options; pass "" to skip tagging (snapshot path).
func (s *BettingService) buildAllMarketViews(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
	bettorDiscordID string,
) ([]marketViewWithID, error) {
	type marketEntry struct {
		marketType string
		title      string
		minPlayers int
		buildFn    func() (*bettingdb.Market, []pricedOption, bool, error)
	}

	entries := []marketEntry{
		{
			marketType: winnerMarketType,
			title:      winnerMarketTitle(round),
			minPlayers: 2,
			buildFn: func() (*bettingdb.Market, []pricedOption, bool, error) {
				return s.getOrBuildWinnerMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
			},
		},
		{
			marketType: placement2ndMarketType,
			title:      placement2ndMarketTitle(round),
			minPlayers: 3,
			buildFn: func() (*bettingdb.Market, []pricedOption, bool, error) {
				return s.getOrBuildPlacement2ndMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
			},
		},
		{
			marketType: placementLastMarketType,
			title:      placementLastMarketTitle(round),
			minPlayers: 3,
			buildFn: func() (*bettingdb.Market, []pricedOption, bool, error) {
				return s.getOrBuildPlacementLastMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
			},
		},
		{
			marketType: placement3rdMarketType,
			title:      placement3rdMarketTitle(round),
			minPlayers: 4,
			buildFn: func() (*bettingdb.Market, []pricedOption, bool, error) {
				return s.getOrBuildPlacement3rdMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
			},
		},
		{
			marketType: overUnderMarketType,
			title:      overUnderMarketTitle(round),
			minPlayers: 2,
			buildFn: func() (*bettingdb.Market, []pricedOption, bool, error) {
				return s.getOrBuildOverUnderMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
			},
		},
	}

	result := make([]marketViewWithID, 0, len(entries))
	for _, e := range entries {
		if len(participants) < e.minPlayers {
			continue
		}
		market, options, ephemeral, err := e.buildFn()
		if err != nil {
			return nil, fmt.Errorf("build market %s: %w", e.marketType, err)
		}
		opts := toAPIOptions(options)
		if bettorDiscordID != "" && marketTypeProhibitsSelfBet(e.marketType) {
			for i := range opts {
				if opts[i].MemberID == bettorDiscordID {
					opts[i].SelfBetRestricted = true
				}
			}
		}
		var dbID int64
		if market != nil {
			dbID = market.ID
		}
		result = append(result, marketViewWithID{
			BettingMarket: BettingMarket{
				ID:        marketIDValue(market),
				Type:      e.marketType,
				Title:     e.title,
				Status:    effectiveMarketStatus(marketStatusValue(market), roundStartTime(round)),
				LocksAt:   roundStartTime(round),
				Ephemeral: ephemeral,
				Result:    marketResultValue(market),
				Options:   opts,
			},
			ID: dbID,
		})
	}
	return result, nil
}
