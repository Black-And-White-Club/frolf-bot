package bettingservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *BettingService) PlaceBet(ctx context.Context, req PlaceBetRequest) (*BetTicket, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "PlaceBet", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.PlaceBet")
		defer span.End()
	}

	if req.ClubUUID == uuid.Nil || req.UserUUID == uuid.Nil {
		s.metrics.RecordOperationFailure(ctx, "PlaceBet", "betting")
		return nil, ErrMembershipRequired
	}
	if req.Stake <= 0 {
		s.metrics.RecordBetRejected(ctx, "invalid_stake")
		s.metrics.RecordOperationFailure(ctx, "PlaceBet", "betting")
		return nil, ErrBetStakeInvalid
	}
	if strings.TrimSpace(req.SelectionKey) == "" {
		s.metrics.RecordBetRejected(ctx, "invalid_selection")
		s.metrics.RecordOperationFailure(ctx, "PlaceBet", "betting")
		return nil, ErrSelectionInvalid
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("betting.club_uuid", req.ClubUUID.String()),
			attribute.String("betting.selection_key", req.SelectionKey),
			attribute.Int("betting.stake", req.Stake),
		)
	}

	var rejectionReason string
	run := func(ctx context.Context, db bun.IDB) (*BetTicket, error) {
		guildID, access, err := s.resolveAccess(ctx, db, req.ClubUUID, req.UserUUID)
		if err != nil {
			return nil, err
		}

		switch access.State {
		case guildtypes.FeatureAccessStateDisabled:
			rejectionReason = "access_denied"
			s.metrics.RecordAccessDenied(ctx, "disabled")
			s.metrics.RecordBetRejected(ctx, "access_denied")
			s.logWarn(ctx, "betting.bet.rejected", "access denied: feature disabled",
				attr.UUIDValue("club_uuid", req.ClubUUID),
			)
			return nil, ErrFeatureDisabled
		case guildtypes.FeatureAccessStateFrozen:
			rejectionReason = "access_denied"
			s.metrics.RecordAccessDenied(ctx, "frozen")
			s.metrics.RecordBetRejected(ctx, "access_denied")
			s.logWarn(ctx, "betting.bet.rejected", "access denied: feature frozen",
				attr.UUIDValue("club_uuid", req.ClubUUID),
			)
			return nil, ErrFeatureFrozen
		}

		// Season-point deltas are now mirrored into the wallet journal via
		// round.points.awarded.v1 — derive seasonID directly without a live
		// leaderboard standing query.
		activeSeason, err := s.leaderboardRepo.GetActiveSeason(ctx, db, string(guildID))
		if err != nil {
			return nil, fmt.Errorf("load active season: %w", err)
		}
		seasonID := defaultSeasonID
		if activeSeason != nil {
			seasonID = activeSeason.ID
		}

		walletBalance, err := s.repo.AcquireWalletBalance(ctx, db, req.ClubUUID, req.UserUUID, seasonID)
		if err != nil {
			return nil, fmt.Errorf("acquire betting wallet lock: %w", err)
		}

		round, err := s.roundRepo.GetRound(ctx, db, guildID, req.RoundID)
		if err != nil {
			return nil, fmt.Errorf("load betting round: %w", err)
		}

		participants, _, err := s.collectEligibleParticipants(ctx, db, req.ClubUUID, round)
		if err != nil {
			return nil, err
		}
		if len(participants) < 2 {
			return nil, ErrNoEligibleRound
		}

		market, options, err := s.dispatchMarket(ctx, db, req.ClubUUID, seasonID, guildID, round, participants, req.MarketType)
		if err != nil {
			return nil, err
		}
		if effectiveMarketStatus(market.Status, market.LocksAt) != openMarketStatus {
			rejectionReason = "market_locked"
			s.metrics.RecordBetRejected(ctx, "market_locked")
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.AddEvent("market.locked")
			}
			s.logWarn(ctx, "betting.bet.rejected", "market is locked",
				attr.UUIDValue("club_uuid", req.ClubUUID),
				attr.String("market_type", market.MarketType),
			)
			return nil, ErrMarketLocked
		}

		selection, ok := findOptionByKey(options, strings.TrimSpace(req.SelectionKey))
		if !ok {
			rejectionReason = "invalid_selection"
			s.metrics.RecordBetRejected(ctx, "invalid_selection")
			return nil, ErrSelectionInvalid
		}

		// Self-bet prevention: blocked for placement and O/U markets.
		if marketTypeProhibitsSelfBet(market.MarketType) {
			bettor, userErr := s.userRepo.GetUserByUUID(ctx, db, req.UserUUID)
			if userErr == nil && bettor != nil && string(bettor.GetUserID()) == string(selection.memberID) {
				rejectionReason = "self_bet_prohibited"
				s.metrics.RecordBetRejected(ctx, "self_bet_prohibited")
				return nil, ErrSelfBetProhibited
			}
		}

		available := walletBalance.Balance - walletBalance.Reserved
		if req.Stake > available {
			rejectionReason = "insufficient_funds"
			s.metrics.RecordBetRejected(ctx, "insufficient_funds")
			s.logWarn(ctx, "betting.bet.rejected", "insufficient funds",
				attr.UUIDValue("club_uuid", req.ClubUUID),
				attr.Int("stake", req.Stake),
				attr.Int("available", available),
			)
			return nil, ErrInsufficientBalance
		}

		bet := &bettingdb.Bet{
			ClubUUID:         req.ClubUUID,
			UserUUID:         req.UserUUID,
			SeasonID:         seasonID,
			RoundID:          req.RoundID.UUID(),
			MarketID:         market.ID,
			MarketType:       market.MarketType,
			SelectionKey:     selection.optionKey,
			SelectionLabel:   selection.label,
			Stake:            req.Stake,
			DecimalOddsCents: selection.decimalOddsCents,
			PotentialPayout:  calculatePotentialPayout(req.Stake, selection.decimalOddsCents),
			Status:           acceptedBetStatus,
		}
		if req.IdempotencyKey != "" {
			bet.IdempotencyKey = &req.IdempotencyKey
		}
		if err := s.repo.CreateBet(ctx, db, bet); err != nil {
			// If the caller supplied an idempotency key and a matching bet already
			// exists, return that existing bet instead of treating it as an error.
			if req.IdempotencyKey != "" {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					var existing bettingdb.Bet
					if qErr := db.NewSelect().
						Model(&existing).
						Where("user_uuid = ?", req.UserUUID).
						Where("idempotency_key = ?", req.IdempotencyKey).
						Scan(ctx); qErr == nil {
						ticket := toTicket(existing)
						return &ticket, nil
					}
				}
			}
			return nil, fmt.Errorf("create betting bet: %w", err)
		}

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.AddEvent("bet.accepted",
				trace.WithAttributes(
					attribute.Int("betting.potential_payout", bet.PotentialPayout),
					attribute.Float64("betting.decimal_odds", decimalOddsFromCents(bet.DecimalOddsCents)),
				),
			)
		}

		entry := &bettingdb.WalletJournalEntry{
			ClubUUID:  req.ClubUUID,
			UserUUID:  req.UserUUID,
			SeasonID:  seasonID,
			EntryType: stakeReservedEntry,
			Amount:    -req.Stake,
			Reason:    fmt.Sprintf("Reserved for %s: %s", market.Title, selection.label),
			CreatedBy: req.UserUUID.String(),
		}
		if err := s.repo.CreateWalletJournalEntry(ctx, db, entry); err != nil {
			return nil, fmt.Errorf("create betting wallet reserve entry: %w", err)
		}

		// Update the wallet balance projection: reserved increases by stake.
		// The lock acquired above is still held; this update is visible to any
		// subsequent transaction that calls AcquireWalletBalance for this user.
		if err := s.repo.ApplyWalletBalanceDelta(ctx, db, req.ClubUUID, req.UserUUID, seasonID, 0, req.Stake); err != nil {
			return nil, fmt.Errorf("update wallet balance reserved: %w", err)
		}

		return &BetTicket{
			ID:              bet.ID,
			RoundID:         round.ID.String(),
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
		}, nil
	}

	ticket, err := runInTx(ctx, s.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted}, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "PlaceBet", "betting")
		if rejectionReason == "" {
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.RecordError(err)
			}
			s.logError(ctx, "betting.operation.failed", "PlaceBet failed", err)
		}
		return nil, err
	}

	s.metrics.RecordBetPlaced(ctx, ticket.MarketType)
	s.metrics.RecordBetStake(ctx, ticket.MarketType, req.Stake)
	s.metrics.RecordOperationSuccess(ctx, "PlaceBet", "betting")
	s.metrics.RecordOperationDuration(ctx, "PlaceBet", "betting", time.Since(start))

	s.logInfo(ctx, "betting.bet.placed", "bet placed",
		attr.UUIDValue("club_uuid", req.ClubUUID),
		attr.String("market_type", ticket.MarketType),
		attr.String("selection_key", req.SelectionKey),
		attr.Int("stake", req.Stake),
		attr.Int("potential_payout", ticket.PotentialPayout),
	)

	return ticket, nil
}

func (s *BettingService) findNextEligibleRound(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	guildID sharedtypes.GuildID,
) (*roundtypes.Round, []targetParticipant, []string, error) {
	rounds, err := s.roundRepo.GetUpcomingRounds(ctx, db, guildID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load upcoming betting rounds: %w", err)
	}
	if len(rounds) == 0 {
		return nil, nil, nil, ErrNoEligibleRound
	}

	sort.Slice(rounds, func(i, j int) bool {
		return roundStartTime(rounds[i]).Before(roundStartTime(rounds[j]))
	})

	for _, round := range rounds {
		participants, warnings, err := s.collectEligibleParticipants(ctx, db, clubUUID, round)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(participants) >= 2 {
			return round, participants, warnings, nil
		}
	}

	return nil, nil, nil, ErrNoEligibleRound
}

func (s *BettingService) collectEligibleParticipants(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	round *roundtypes.Round,
) ([]targetParticipant, []string, error) {
	eligible := make([]targetParticipant, 0, len(round.Participants))
	optedOutCount := 0

	for _, participant := range round.Participants {
		if participant.Response != roundtypes.ResponseAccept || participant.UserID == "" {
			continue
		}

		userUUID, err := s.userRepo.GetUUIDByDiscordID(ctx, db, participant.UserID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				continue
			}
			return nil, nil, fmt.Errorf("resolve betting participant uuid: %w", err)
		}

		setting, err := s.repo.GetMemberSettings(ctx, db, clubUUID, userUUID)
		if err != nil {
			return nil, nil, fmt.Errorf("load betting participant settings: %w", err)
		}
		if setting != nil && setting.OptOutTargeting {
			optedOutCount++
			continue
		}

		eligible = append(eligible, targetParticipant{
			participant: participant,
			userUUID:    userUUID,
			label:       s.resolveParticipantLabel(ctx, db, userUUID, participant),
		})
	}

	warnings := make([]string, 0, 1)
	if optedOutCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d accepted player(s) opted out of betting targets for this market.", optedOutCount))
	}

	return eligible, warnings, nil
}

func (s *BettingService) resolveParticipantLabel(
	ctx context.Context,
	db bun.IDB,
	userUUID uuid.UUID,
	participant roundtypes.Participant,
) string {
	if user, err := s.userRepo.GetUserByUUID(ctx, db, userUUID); err == nil && user != nil {
		return user.GetDisplayName()
	}
	if participant.RawName != "" {
		return participant.RawName
	}
	if participant.UserID != "" {
		return string(participant.UserID)
	}
	return userUUID.String()
}

// dispatchMarket resolves the correct ensure function for the requested market
// type, defaulting to winnerMarketType when MarketType is empty.
func (s *BettingService) dispatchMarket(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
	seasonID string,
	guildID sharedtypes.GuildID,
	round *roundtypes.Round,
	participants []targetParticipant,
	marketType string,
) (*bettingdb.Market, []pricedOption, error) {
	if marketType == "" {
		marketType = winnerMarketType
	}
	switch marketType {
	case winnerMarketType:
		m, opts, _, err := s.ensureWinnerMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		return m, opts, err
	case placement2ndMarketType:
		if len(participants) < 3 {
			return nil, nil, ErrNoEligibleRound
		}
		m, opts, _, err := s.ensurePlacement2ndMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		return m, opts, err
	case placement3rdMarketType:
		if len(participants) < 4 {
			return nil, nil, ErrNoEligibleRound
		}
		m, opts, _, err := s.ensurePlacement3rdMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		return m, opts, err
	case placementLastMarketType:
		if len(participants) < 3 {
			return nil, nil, ErrNoEligibleRound
		}
		m, opts, _, err := s.ensurePlacementLastMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		return m, opts, err
	case overUnderMarketType:
		m, opts, _, err := s.ensureOverUnderMarket(ctx, db, clubUUID, seasonID, guildID, round, participants)
		return m, opts, err
	default:
		return nil, nil, ErrInvalidMarketType
	}
}

// logInfo emits a structured info log including trace/span IDs for Grafana correlation.
func (s *BettingService) logInfo(ctx context.Context, event, msg string, attrs ...slog.Attr) {
	traceAttrs := attr.TraceContext(ctx)
	args := make([]any, 0, len(traceAttrs)+len(attrs)+1)
	args = append(args, attr.String("event", event))
	for _, a := range traceAttrs {
		args = append(args, a)
	}
	for _, a := range attrs {
		args = append(args, a)
	}
	s.logger.InfoContext(ctx, msg, args...)
}

// logWarn emits a structured warn log including trace/span IDs.
func (s *BettingService) logWarn(ctx context.Context, event, msg string, attrs ...slog.Attr) {
	traceAttrs := attr.TraceContext(ctx)
	args := make([]any, 0, len(traceAttrs)+len(attrs)+1)
	args = append(args, attr.String("event", event))
	for _, a := range traceAttrs {
		args = append(args, a)
	}
	for _, a := range attrs {
		args = append(args, a)
	}
	s.logger.WarnContext(ctx, msg, args...)
}

// logError emits a structured error log including trace/span IDs.
func (s *BettingService) logError(ctx context.Context, event, msg string, err error, attrs ...slog.Attr) {
	traceAttrs := attr.TraceContext(ctx)
	args := make([]any, 0, len(traceAttrs)+len(attrs)+2)
	args = append(args, attr.String("event", event), attr.Error(err))
	for _, a := range traceAttrs {
		args = append(args, a)
	}
	for _, a := range attrs {
		args = append(args, a)
	}
	s.logger.ErrorContext(ctx, msg, args...)
}
