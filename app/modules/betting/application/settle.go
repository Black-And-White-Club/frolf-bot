package bettingservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (s *BettingService) SettleRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	round *BettingSettlementRound,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) ([]MarketSettlementResult, error) {
	if round == nil {
		return nil, nil
	}

	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "SettleRound", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.SettleRound")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.round_id", round.ID.String()),
			attribute.String("betting.guild_id", string(guildID)),
			attribute.String("betting.settlement_source", source),
		)
	}

	run := func(ctx context.Context, db bun.IDB) ([]MarketSettlementResult, error) {
		clubUUID, err := s.userRepo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve club uuid by guild id: %w", err)
		}

		// Settlement invariants:
		//   - FROZEN: settlement must continue so that accepted tickets can be
		//     resolved and wallets credited even when new bets are blocked.
		//   - DISABLED (manual_deny): the operator has fully blocked the feature;
		//     settlement is also blocked to avoid crediting a non-existent wallet.
		//   - DISABLED (other source): same as manual_deny — treat consistently.
		entitlements, err := s.guildRepo.ResolveEntitlements(ctx, db, guildID)
		if err != nil {
			return nil, fmt.Errorf("resolve entitlements for guild %s: %w", guildID, err)
		}
		bettingAccess := entitlements.Features[guildtypes.ClubFeatureBetting]
		if bettingAccess.State == guildtypes.FeatureAccessStateDisabled {
			return nil, ErrFeatureDisabled
		}

		markets, err := s.repo.ListMarketsByRound(ctx, db, clubUUID, round.ID.UUID())
		if err != nil {
			return nil, fmt.Errorf("load betting markets by round: %w", err)
		}

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(attribute.Int("betting.market_count", len(markets)))
		}

		var results []MarketSettlementResult
		for idx := range markets {
			if _, err := s.settleMarket(ctx, db, &markets[idx], round, source, actorUUID, reason); err != nil {
				return nil, err
			}
			results = append(results, MarketSettlementResult{
				GuildID:           guildID,
				ClubUUID:          clubUUID.String(),
				RoundID:           round.ID,
				MarketID:          markets[idx].ID,
				ResultSummary:     markets[idx].ResultSummary,
				SettlementVersion: markets[idx].SettlementVersion,
			})
		}
		return results, nil
	}

	const maxSettleRetries = 3
	var results []MarketSettlementResult
	var err error
	for attempt := range maxSettleRetries {
		results, err = runInTx(ctx, s.db, &sql.TxOptions{Isolation: sql.LevelSerializable}, run)
		if err == nil {
			break
		}
		if !isSerializationFailure(err) || attempt == maxSettleRetries-1 {
			break
		}
		backoff := time.Duration(50*(1<<attempt)) * time.Millisecond // 50ms, 100ms, 200ms
		s.logger.WarnContext(ctx, "settlement serialization conflict, retrying",
			attr.Int("attempt", attempt+1),
			attr.String("backoff", backoff.String()),
		)
		time.Sleep(backoff)
	}
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "SettleRound", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.operation.failed", "SettleRound failed", err,
			attr.String("guild_id", string(guildID)),
			attr.String("settlement_source", source),
		)
		return nil, err
	}

	s.metrics.RecordRoundSettled(ctx, source)
	s.metrics.RecordOperationSuccess(ctx, "SettleRound", "betting")
	s.metrics.RecordOperationDuration(ctx, "SettleRound", "betting", time.Since(start))
	s.metrics.RecordSettlementDuration(ctx, source, time.Since(start))

	s.logInfo(ctx, "betting.round.settled", "round settled",
		attr.String("guild_id", string(guildID)),
		attr.String("settlement_source", source),
		attr.Int("market_count", len(results)),
	)

	return results, nil
}

func (s *BettingService) settleMarket(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	round *BettingSettlementRound,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) (bool, error) {
	switch market.MarketType {
	case winnerMarketType:
		return s.settleWinnerMarket(ctx, db, market, round, source, actorUUID, reason)
	case placement2ndMarketType:
		return s.settlePlacementMarket(ctx, db, market, round, source, actorUUID, reason, 2)
	case placement3rdMarketType:
		return s.settlePlacementMarket(ctx, db, market, round, source, actorUUID, reason, 3)
	case placementLastMarketType:
		return s.settlePlacementMarket(ctx, db, market, round, source, actorUUID, reason, -1) // -1 = dynamic last
	case overUnderMarketType:
		return s.settleOverUnderMarket(ctx, db, market, round, source, actorUUID, reason)
	default:
		return false, fmt.Errorf("unsupported market type %q", market.MarketType)
	}
}

func (s *BettingService) settleWinnerMarket(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	round *BettingSettlementRound,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) (bool, error) {
	options, err := s.repo.ListMarketOptions(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market options: %w", err)
	}
	bets, err := s.repo.ListBetsForMarket(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market bets: %w", err)
	}

	outcome := deriveWinnerOutcome(round, options)
	if outcome.status == voidedMarketStatus {
		return s.applyVoidSettlement(ctx, db, market, bets, actorUUID, blankIfEmpty(reason, outcome.voidReason), source, true)
	}

	now := time.Now().UTC()
	changed := false
	wonCount, lostCount, voidedCount := 0, 0, 0

	for idx := range bets {
		decision := decideWinnerBetOutcome(bets[idx], outcome)
		if !betNeedsUpdate(&bets[idx], decision.status, decision.payout) {
			continue
		}
		delta := decision.payout - bets[idx].SettledPayout
		if delta != 0 {
			entryType := marketSettlementEntry
			entryReason := decision.reason
			if decision.status == voidedBetStatus {
				entryType = marketRefundEntry
			} else if delta < 0 {
				entryType = marketCorrectionEntry
			}
			if err := s.repo.CreateWalletJournalEntry(ctx, db, &bettingdb.WalletJournalEntry{
				ClubUUID:  bets[idx].ClubUUID,
				UserUUID:  bets[idx].UserUUID,
				SeasonID:  bets[idx].SeasonID,
				EntryType: entryType,
				Amount:    delta,
				Reason:    entryReason,
				CreatedBy: createdByValue(actorUUID, source),
			}); err != nil {
				return false, fmt.Errorf("create settlement journal entry: %w", err)
			}
		}

		// Keep the wallet balance projection in sync.
		// balance += amount credited to the journal (delta); reserved decreases
		// by the full stake since the bet is leaving the accepted state.
		balanceDelta := delta
		reservedDelta := 0
		if bets[idx].Status == acceptedBetStatus {
			reservedDelta = -bets[idx].Stake
		}
		if balanceDelta != 0 || reservedDelta != 0 {
			if err := s.repo.ApplyWalletBalanceDelta(ctx, db, bets[idx].ClubUUID, bets[idx].UserUUID, bets[idx].SeasonID, balanceDelta, reservedDelta); err != nil {
				return false, fmt.Errorf("update wallet balance on settlement: %w", err)
			}
		}

		bets[idx].Status = decision.status
		bets[idx].SettledPayout = decision.payout
		bets[idx].SettledAt = &now
		if err := s.repo.UpdateBet(ctx, db, &bets[idx]); err != nil {
			return false, fmt.Errorf("update bet settlement: %w", err)
		}

		s.metrics.RecordBetSettled(ctx, market.MarketType, decision.status)
		if decision.payout > 0 {
			s.metrics.RecordBetPayout(ctx, market.MarketType, decision.payout)
		}

		switch decision.status {
		case wonBetStatus:
			wonCount++
		case lostBetStatus:
			lostCount++
		case voidedBetStatus:
			voidedCount++
		}
		changed = true
	}

	if !marketNeedsUpdate(market, settledMarketStatus, outcome.resolvedOptionKeys, "", outcome.summary, source) && !changed {
		return false, nil
	}

	market.Status = settledMarketStatus
	market.ResolvedOptionKey = outcome.resolvedOptionKeys
	market.VoidReason = ""
	market.ResultSummary = outcome.summary
	market.LastResultSource = source
	market.SettledAt = &now
	market.SettlementVersion++
	market.UpdatedAt = now
	if err := s.repo.UpdateMarket(ctx, db, market); err != nil {
		if errors.Is(err, bettingdb.ErrSettlementVersionConflict) {
			return false, nil
		}
		return false, fmt.Errorf("update market settlement: %w", err)
	}

	s.metrics.RecordMarketSettled(ctx, market.MarketType)

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.AddEvent("settlement.decision",
			trace.WithAttributes(
				attribute.Int("betting.won_count", wonCount),
				attribute.Int("betting.lost_count", lostCount),
				attribute.Int("betting.voided_count", voidedCount),
			),
		)
	}

	if err := s.repo.CreateAuditLog(ctx, db, &bettingdb.AuditLog{
		ClubUUID:      market.ClubUUID,
		MarketID:      int64Ptr(market.ID),
		RoundID:       uuidPtr(market.RoundID),
		ActorUserUUID: actorUUID,
		Action:        "market_settled",
		Reason:        blankIfEmpty(reason, outcome.summary),
		Metadata:      fmt.Sprintf("source=%s result=%s", source, outcome.summary),
	}); err != nil {
		return false, fmt.Errorf("create settlement audit log: %w", err)
	}

	s.logInfo(ctx, "betting.market.settled", "market settled",
		attr.String("market_type", market.MarketType),
		attr.Int("won", wonCount),
		attr.Int("lost", lostCount),
		attr.Int("voided", voidedCount),
		attr.String("result_summary", outcome.summary),
	)

	return true, nil
}

// deriveWinnerOutcome determines the settlement outcome for a winner market by
// comparing round finalization data against the persisted market options.
func deriveWinnerOutcome(round *BettingSettlementRound, options []bettingdb.MarketOption) winnerOutcome {
	participants := make(map[string]BettingSettlementParticipant, len(round.Participants))
	for _, participant := range round.Participants {
		if participant.MemberID == "" {
			continue
		}
		participants[participant.MemberID] = participant
	}

	scratched := make(map[string]struct{})
	optionLabels := make(map[string]string, len(options))
	for _, option := range options {
		optionLabels[option.OptionKey] = option.Label
		participant, ok := participants[option.ParticipantMemberID]
		if !ok || !strings.EqualFold(participant.Response, string(roundtypes.ResponseAccept)) || (participant.Score == nil && !participant.IsDNF) {
			scratched[option.OptionKey] = struct{}{}
		}
	}

	bestScore := int(^uint(0) >> 1)
	winningKeys := make([]string, 0, len(options))
	for _, option := range options {
		participant, ok := participants[option.ParticipantMemberID]
		if !ok || participant.Score == nil || participant.IsDNF || !strings.EqualFold(participant.Response, string(roundtypes.ResponseAccept)) {
			continue
		}
		if *participant.Score < bestScore {
			bestScore = *participant.Score
			winningKeys = []string{option.OptionKey}
			continue
		}
		if *participant.Score == bestScore {
			winningKeys = append(winningKeys, option.OptionKey)
		}
	}

	if len(winningKeys) == 0 {
		return winnerOutcome{
			status:     voidedMarketStatus,
			voidReason: "No scored finishers were available for settlement.",
			scratched:  scratched,
		}
	}

	sort.Strings(winningKeys)
	winners := make(map[string]struct{}, len(winningKeys))
	winnerLabels := make([]string, 0, len(winningKeys))
	for _, key := range winningKeys {
		winners[key] = struct{}{}
		if label := optionLabels[key]; label != "" {
			winnerLabels = append(winnerLabels, label)
		} else {
			winnerLabels = append(winnerLabels, key)
		}
	}

	summary := ""
	if len(winnerLabels) == 1 {
		summary = fmt.Sprintf("%s won the round.", winnerLabels[0])
	} else {
		summary = fmt.Sprintf("Tied winners: %s.", strings.Join(winnerLabels, ", "))
	}

	return winnerOutcome{
		status:             settledMarketStatus,
		resolvedOptionKeys: strings.Join(winningKeys, ","),
		summary:            summary,
		winners:            winners,
		scratched:          scratched,
	}
}

func decideWinnerBetOutcome(bet bettingdb.Bet, outcome winnerOutcome) settlementDecision {
	if _, scratched := outcome.scratched[bet.SelectionKey]; scratched {
		return settlementDecision{
			status: voidedBetStatus,
			payout: bet.Stake,
			reason: fmt.Sprintf("Refunded: %s was not an active finisher for this market.", bet.SelectionLabel),
		}
	}
	if _, won := outcome.winners[bet.SelectionKey]; won {
		return settlementDecision{
			status: wonBetStatus,
			payout: bet.PotentialPayout,
			reason: fmt.Sprintf("Settled winning ticket for %s.", bet.SelectionLabel),
		}
	}
	return settlementDecision{
		status: lostBetStatus,
		payout: 0,
		reason: fmt.Sprintf("Settled losing ticket for %s.", bet.SelectionLabel),
	}
}

// settlementRoundFromRound converts a round.Round into the BettingSettlementRound
// projection used by the settlement logic.
func settlementRoundFromRound(round *roundtypes.Round) *BettingSettlementRound {
	if round == nil {
		return nil
	}

	participants := make([]BettingSettlementParticipant, 0, len(round.Participants))
	for _, participant := range round.Participants {
		var score *int
		if participant.Score != nil {
			v := int(*participant.Score)
			score = &v
		}
		participants = append(participants, BettingSettlementParticipant{
			MemberID: string(participant.UserID),
			Response: string(participant.Response),
			Score:    score,
			IsDNF:    participant.IsDNF,
		})
	}

	return &BettingSettlementRound{
		ID:           round.ID,
		Title:        round.Title.String(),
		GuildID:      round.GuildID,
		Finalized:    bool(round.Finalized) || round.State == roundtypes.RoundStateFinalized,
		Participants: participants,
	}
}

// isSerializationFailure reports whether err represents a PostgreSQL
// serialization failure (error code "40001"). These are transient and can be
// safely retried.
func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

// ---------------------------------------------------------------------------
// Placement market settlement
// ---------------------------------------------------------------------------

// settlePlacementMarket settles exact-position markets (2nd, 3rd, last).
// targetPosition is 1-indexed; pass -1 to indicate "last" (dynamic: count of
// scored finishers).
func (s *BettingService) settlePlacementMarket(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	round *BettingSettlementRound,
	source string,
	actorUUID *uuid.UUID,
	reason string,
	targetPosition int,
) (bool, error) {
	options, err := s.repo.ListMarketOptions(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market options: %w", err)
	}
	bets, err := s.repo.ListBetsForMarket(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market bets: %w", err)
	}

	outcome := derivePlacementOutcome(round, options, targetPosition)
	if outcome.status == voidedMarketStatus {
		return s.applyVoidSettlement(ctx, db, market, bets, actorUUID, blankIfEmpty(reason, outcome.voidReason), source, true)
	}

	return s.applySettlementDecisions(ctx, db, market, bets, outcome, source, actorUUID, reason)
}

// derivePlacementOutcome ranks scored participants and returns which option
// keys correspond to the target position.
func derivePlacementOutcome(round *BettingSettlementRound, options []bettingdb.MarketOption, targetPosition int) winnerOutcome {
	participants := make(map[string]BettingSettlementParticipant, len(round.Participants))
	for _, p := range round.Participants {
		if p.MemberID != "" {
			participants[p.MemberID] = p
		}
	}

	scratched := make(map[string]struct{})
	optionLabels := make(map[string]string, len(options))
	for _, opt := range options {
		optionLabels[opt.OptionKey] = opt.Label
		p, ok := participants[opt.ParticipantMemberID]
		if !ok || !strings.EqualFold(p.Response, string(roundtypes.ResponseAccept)) || p.Score == nil || p.IsDNF {
			scratched[opt.OptionKey] = struct{}{}
		}
	}

	// Collect and rank scored finishers (ascending score = best).
	type scored struct {
		memberID string
		score    int
	}
	var finishers []scored
	for _, opt := range options {
		p, ok := participants[opt.ParticipantMemberID]
		if !ok || p.Score == nil || p.IsDNF || !strings.EqualFold(p.Response, string(roundtypes.ResponseAccept)) {
			continue
		}
		finishers = append(finishers, scored{memberID: opt.ParticipantMemberID, score: *p.Score})
	}
	// Deduplicate by memberID.
	seen := make(map[string]struct{})
	unique := finishers[:0]
	for _, f := range finishers {
		if _, dup := seen[f.memberID]; !dup {
			seen[f.memberID] = struct{}{}
			unique = append(unique, f)
		}
	}
	finishers = unique

	if len(finishers) == 0 {
		return winnerOutcome{
			status:     voidedMarketStatus,
			voidReason: "No scored finishers were available for settlement.",
			scratched:  scratched,
		}
	}

	// Sort ascending (lower score = better).
	sort.Slice(finishers, func(i, j int) bool { return finishers[i].score < finishers[j].score })

	// Resolve effective target: -1 means last place.
	effectiveTarget := targetPosition
	if targetPosition == -1 {
		effectiveTarget = len(finishers)
	}

	if effectiveTarget > len(finishers) {
		return winnerOutcome{
			status:     voidedMarketStatus,
			voidReason: fmt.Sprintf("Not enough scored finishers for position %d.", effectiveTarget),
			scratched:  scratched,
		}
	}

	// Find the score at the target rank (0-indexed).
	targetScore := finishers[effectiveTarget-1].score

	// All members with that score share the target position.
	winnerIDs := make(map[string]struct{})
	for _, f := range finishers {
		if f.score == targetScore {
			winnerIDs[f.memberID] = struct{}{}
		}
	}

	// Map winning member IDs → winning option keys.
	winningKeys := make([]string, 0)
	winnerLabels := make([]string, 0)
	winners := make(map[string]struct{})
	for _, opt := range options {
		if _, ok := winnerIDs[opt.ParticipantMemberID]; ok {
			winningKeys = append(winningKeys, opt.OptionKey)
			winners[opt.OptionKey] = struct{}{}
			if lbl := optionLabels[opt.OptionKey]; lbl != "" {
				winnerLabels = append(winnerLabels, lbl)
			}
		}
	}
	sort.Strings(winningKeys)

	summary := ""
	if len(winnerLabels) == 1 {
		summary = fmt.Sprintf("%s finished %s.", winnerLabels[0], ordinal(effectiveTarget))
	} else {
		summary = fmt.Sprintf("Tied %s: %s.", ordinal(effectiveTarget), strings.Join(winnerLabels, ", "))
	}

	return winnerOutcome{
		status:             settledMarketStatus,
		resolvedOptionKeys: strings.Join(winningKeys, ","),
		summary:            summary,
		winners:            winners,
		scratched:          scratched,
	}
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// ---------------------------------------------------------------------------
// Over/Under market settlement
// ---------------------------------------------------------------------------

// settleOverUnderMarket settles per-player score over/under markets.
func (s *BettingService) settleOverUnderMarket(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	round *BettingSettlementRound,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) (bool, error) {
	options, err := s.repo.ListMarketOptions(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market options: %w", err)
	}
	bets, err := s.repo.ListBetsForMarket(ctx, db, market.ID)
	if err != nil {
		return false, fmt.Errorf("load market bets: %w", err)
	}

	outcome := deriveOverUnderOutcome(round, options)
	if outcome.status == voidedMarketStatus {
		return s.applyVoidSettlement(ctx, db, market, bets, actorUUID, blankIfEmpty(reason, outcome.voidReason), source, true)
	}

	return s.applySettlementDecisions(ctx, db, market, bets, outcome, source, actorUUID, reason)
}

// deriveOverUnderOutcome determines winning option keys for an O/U market.
// Over wins when actual score > line; under wins when actual score <= line (under on push).
func deriveOverUnderOutcome(round *BettingSettlementRound, options []bettingdb.MarketOption) winnerOutcome {
	participants := make(map[string]BettingSettlementParticipant, len(round.Participants))
	for _, p := range round.Participants {
		if p.MemberID != "" {
			participants[p.MemberID] = p
		}
	}

	// Group options by participant member ID.
	type playerOptions struct {
		overKey  string
		underKey string
		overLbl  string
		underLbl string
		metadata string
	}
	byMember := make(map[string]*playerOptions)
	for _, opt := range options {
		po := byMember[opt.ParticipantMemberID]
		if po == nil {
			po = &playerOptions{metadata: opt.Metadata}
			byMember[opt.ParticipantMemberID] = po
		}
		if strings.HasSuffix(opt.OptionKey, "_over") {
			po.overKey = opt.OptionKey
			po.overLbl = opt.Label
		} else if strings.HasSuffix(opt.OptionKey, "_under") {
			po.underKey = opt.OptionKey
			po.underLbl = opt.Label
		}
	}

	scratched := make(map[string]struct{})
	winners := make(map[string]struct{})
	winnerLabels := make([]string, 0)
	winningKeys := make([]string, 0)

	for memberID, po := range byMember {
		p, ok := participants[memberID]
		isDNF := !ok || p.IsDNF || p.Score == nil || !strings.EqualFold(p.Response, string(roundtypes.ResponseAccept))
		if isDNF {
			if po.overKey != "" {
				scratched[po.overKey] = struct{}{}
			}
			if po.underKey != "" {
				scratched[po.underKey] = struct{}{}
			}
			continue
		}

		line := parseLineFromMetadata(po.metadata)
		actualScore := *p.Score

		if actualScore > line {
			// Over wins.
			if po.overKey != "" {
				winners[po.overKey] = struct{}{}
				winningKeys = append(winningKeys, po.overKey)
				winnerLabels = append(winnerLabels, po.overLbl)
			}
		} else {
			// Under wins (includes push: score == line).
			if po.underKey != "" {
				winners[po.underKey] = struct{}{}
				winningKeys = append(winningKeys, po.underKey)
				winnerLabels = append(winnerLabels, po.underLbl)
			}
		}
	}

	if len(winners) == 0 && len(scratched) == len(options) {
		return winnerOutcome{
			status:     voidedMarketStatus,
			voidReason: "All players were scratched from the over/under market.",
			scratched:  scratched,
		}
	}

	sort.Strings(winningKeys)
	summary := fmt.Sprintf("O/U settled: %s.", strings.Join(winnerLabels, "; "))

	return winnerOutcome{
		status:             settledMarketStatus,
		resolvedOptionKeys: strings.Join(winningKeys, ","),
		summary:            summary,
		winners:            winners,
		scratched:          scratched,
	}
}

// parseLineFromMetadata extracts the integer line value from JSON metadata
// like {"line":52}. Returns 0 if parsing fails.
func parseLineFromMetadata(metadata string) int {
	// Simple manual parse to avoid importing encoding/json in the hot path.
	// Metadata format is always produced by our own code as {"line":N}.
	idx := strings.Index(metadata, `"line":`)
	if idx < 0 {
		return 0
	}
	rest := strings.TrimSpace(metadata[idx+7:])
	line := 0
	for _, c := range rest {
		if c >= '0' && c <= '9' {
			line = line*10 + int(c-'0')
		} else if c == '-' && line == 0 {
			continue
		} else {
			break
		}
	}
	return line
}

// ---------------------------------------------------------------------------
// Shared settlement application helpers
// ---------------------------------------------------------------------------

// applySettlementDecisions applies won/lost/voided outcomes to each bet and
// updates the market record. It is reused by winner, placement, and O/U settlement.
func (s *BettingService) applySettlementDecisions(
	ctx context.Context,
	db bun.IDB,
	market *bettingdb.Market,
	bets []bettingdb.Bet,
	outcome winnerOutcome,
	source string,
	actorUUID *uuid.UUID,
	reason string,
) (bool, error) {
	now := time.Now().UTC()
	changed := false
	wonCount, lostCount, voidedCount := 0, 0, 0

	for idx := range bets {
		decision := decideWinnerBetOutcome(bets[idx], outcome)
		if !betNeedsUpdate(&bets[idx], decision.status, decision.payout) {
			continue
		}
		delta := decision.payout - bets[idx].SettledPayout
		if delta != 0 {
			entryType := marketSettlementEntry
			entryReason := decision.reason
			if decision.status == voidedBetStatus {
				entryType = marketRefundEntry
			} else if delta < 0 {
				entryType = marketCorrectionEntry
			}
			if err := s.repo.CreateWalletJournalEntry(ctx, db, &bettingdb.WalletJournalEntry{
				ClubUUID:  bets[idx].ClubUUID,
				UserUUID:  bets[idx].UserUUID,
				SeasonID:  bets[idx].SeasonID,
				EntryType: entryType,
				Amount:    delta,
				Reason:    entryReason,
				CreatedBy: createdByValue(actorUUID, source),
			}); err != nil {
				return false, fmt.Errorf("create settlement journal entry: %w", err)
			}
		}

		balanceDelta := delta
		reservedDelta := 0
		if bets[idx].Status == acceptedBetStatus {
			reservedDelta = -bets[idx].Stake
		}
		if balanceDelta != 0 || reservedDelta != 0 {
			if err := s.repo.ApplyWalletBalanceDelta(ctx, db, bets[idx].ClubUUID, bets[idx].UserUUID, bets[idx].SeasonID, balanceDelta, reservedDelta); err != nil {
				return false, fmt.Errorf("update wallet balance on settlement: %w", err)
			}
		}

		bets[idx].Status = decision.status
		bets[idx].SettledPayout = decision.payout
		bets[idx].SettledAt = &now
		if err := s.repo.UpdateBet(ctx, db, &bets[idx]); err != nil {
			return false, fmt.Errorf("update bet settlement: %w", err)
		}

		s.metrics.RecordBetSettled(ctx, market.MarketType, decision.status)
		if decision.payout > 0 {
			s.metrics.RecordBetPayout(ctx, market.MarketType, decision.payout)
		}

		switch decision.status {
		case wonBetStatus:
			wonCount++
		case lostBetStatus:
			lostCount++
		case voidedBetStatus:
			voidedCount++
		}
		changed = true
	}

	if !marketNeedsUpdate(market, settledMarketStatus, outcome.resolvedOptionKeys, "", outcome.summary, source) && !changed {
		return false, nil
	}

	market.Status = settledMarketStatus
	market.ResolvedOptionKey = outcome.resolvedOptionKeys
	market.VoidReason = ""
	market.ResultSummary = outcome.summary
	market.LastResultSource = source
	market.SettledAt = &now
	market.SettlementVersion++
	market.UpdatedAt = now
	if err := s.repo.UpdateMarket(ctx, db, market); err != nil {
		if errors.Is(err, bettingdb.ErrSettlementVersionConflict) {
			return false, nil
		}
		return false, fmt.Errorf("update market settlement: %w", err)
	}

	s.metrics.RecordMarketSettled(ctx, market.MarketType)

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.AddEvent("settlement.decision",
			trace.WithAttributes(
				attribute.Int("betting.won_count", wonCount),
				attribute.Int("betting.lost_count", lostCount),
				attribute.Int("betting.voided_count", voidedCount),
			),
		)
	}

	if err := s.repo.CreateAuditLog(ctx, db, &bettingdb.AuditLog{
		ClubUUID:      market.ClubUUID,
		MarketID:      int64Ptr(market.ID),
		RoundID:       uuidPtr(market.RoundID),
		ActorUserUUID: actorUUID,
		Action:        "market_settled",
		Reason:        blankIfEmpty(reason, outcome.summary),
		Metadata:      fmt.Sprintf("source=%s result=%s", source, outcome.summary),
	}); err != nil {
		return false, fmt.Errorf("create settlement audit log: %w", err)
	}

	s.logInfo(ctx, "betting.market.settled", "market settled",
		attr.String("market_type", market.MarketType),
		attr.Int("won", wonCount),
		attr.Int("lost", lostCount),
		attr.Int("voided", voidedCount),
		attr.String("result_summary", outcome.summary),
	)

	return true, nil
}
