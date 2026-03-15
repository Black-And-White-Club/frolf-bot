package bettingservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (s *BettingService) AdminMarketAction(ctx context.Context, req AdminMarketActionRequest) (*AdminMarketActionResult, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "AdminMarketAction", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.AdminMarketAction")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.admin_action", req.Action),
			attribute.Int64("betting.market_id", req.MarketID),
		)
	}

	if req.ClubUUID == uuid.Nil || req.AdminUUID == uuid.Nil {
		s.metrics.RecordOperationFailure(ctx, "AdminMarketAction", "betting")
		return nil, ErrAdminRequired
	}
	if req.MarketID <= 0 {
		s.metrics.RecordOperationFailure(ctx, "AdminMarketAction", "betting")
		return nil, ErrMarketNotFound
	}
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		s.metrics.RecordOperationFailure(ctx, "AdminMarketAction", "betting")
		return nil, ErrAdjustmentReasonRequired
	}
	if req.Action != adminActionVoid && req.Action != adminActionResettle {
		s.metrics.RecordOperationFailure(ctx, "AdminMarketAction", "betting")
		return nil, ErrInvalidMarketAction
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("betting.action", req.Action),
			attribute.Int64("betting.market_id", req.MarketID),
		)
	}

	run := func(ctx context.Context, db bun.IDB) (*AdminMarketActionResult, error) {
		guildID, _, err := s.resolveAdminAccess(ctx, db, req.ClubUUID, req.AdminUUID)
		if err != nil {
			return nil, err
		}

		market, err := s.repo.GetMarketByID(ctx, db, req.ClubUUID, req.MarketID)
		if err != nil {
			return nil, fmt.Errorf("load betting market: %w", err)
		}
		if market == nil {
			return nil, ErrMarketNotFound
		}

		affected := []int64{market.ID}
		switch req.Action {
		case adminActionVoid:
			if _, err := s.voidMarket(ctx, db, market, &req.AdminUUID, req.Reason, "admin:void"); err != nil {
				return nil, err
			}
		case adminActionResettle:
			round, err := s.roundRepo.GetRound(ctx, db, guildID, sharedtypes.RoundID(market.RoundID))
			if err != nil {
				return nil, fmt.Errorf("load betting round for resettle: %w", err)
			}
			if round == nil || (!bool(round.Finalized) && round.State != roundtypes.RoundStateFinalized) {
				return nil, ErrRoundNotFinalized
			}
			settlementRound := settlementRoundFromRound(round)
			if _, err := s.settleMarket(ctx, db, market, settlementRound, "admin:resettle", &req.AdminUUID, req.Reason); err != nil {
				return nil, err
			}
		}

		return &AdminMarketActionResult{
			MarketID:          market.ID,
			Action:            req.Action,
			Status:            effectiveMarketStatus(market.Status, market.LocksAt),
			ResultSummary:     market.ResultSummary,
			SettlementVersion: market.SettlementVersion,
			SettledAt:         market.SettledAt,
			AffectedMarketIDs: affected,
		}, nil
	}

	result, err := runInTx(ctx, s.db, &sql.TxOptions{Isolation: sql.LevelSerializable}, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "AdminMarketAction", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.admin.market.action.failed", "AdminMarketAction failed", err,
			attr.String("action", req.Action),
			attr.Int("market_id", int(req.MarketID)),
		)
		return nil, err
	}

	s.metrics.RecordAdminMarketAction(ctx, req.Action)
	s.metrics.RecordOperationSuccess(ctx, "AdminMarketAction", "betting")
	s.metrics.RecordOperationDuration(ctx, "AdminMarketAction", "betting", time.Since(start))

	s.logInfo(ctx, "betting.admin.market.action.completed", "admin market action completed",
		attr.String("action", req.Action),
		attr.Int("market_id", int(req.MarketID)),
	)

	return result, nil
}

// resolveAccess verifies club membership for userUUID and returns the guildID
// and the betting feature access state for that guild.
func (s *BettingService) resolveAccess(
	ctx context.Context,
	db bun.IDB,
	clubUUID, userUUID uuid.UUID,
) (sharedtypes.GuildID, guildtypes.ClubFeatureAccess, error) {
	if _, err := s.userRepo.GetClubMembership(ctx, db, userUUID, clubUUID); err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return "", guildtypes.ClubFeatureAccess{}, ErrMembershipRequired
		}
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("load club membership: %w", err)
	}

	guildID, err := s.userRepo.GetDiscordGuildIDByClubUUID(ctx, db, clubUUID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return "", guildtypes.ClubFeatureAccess{}, ErrMembershipRequired
		}
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("resolve club guild id: %w", err)
	}

	entitlements, err := s.guildRepo.ResolveEntitlements(ctx, db, guildID)
	if err != nil {
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("resolve betting entitlements: %w", err)
	}

	feature, ok := entitlements.Feature(guildtypes.ClubFeatureBetting)
	if !ok {
		feature = guildtypes.ClubFeatureAccess{
			Key:    guildtypes.ClubFeatureBetting,
			State:  guildtypes.FeatureAccessStateDisabled,
			Source: guildtypes.FeatureAccessSourceNone,
		}
	}

	return guildID, feature, nil
}

// resolveAccessByClub resolves the guildID and feature access for a club UUID
// without requiring a user membership check. Used for public/NATS snapshot
// endpoints that are not user-specific.
func (s *BettingService) resolveAccessByClub(
	ctx context.Context,
	db bun.IDB,
	clubUUID uuid.UUID,
) (sharedtypes.GuildID, guildtypes.ClubFeatureAccess, error) {
	guildID, err := s.userRepo.GetDiscordGuildIDByClubUUID(ctx, db, clubUUID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return "", guildtypes.ClubFeatureAccess{}, ErrMembershipRequired
		}
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("resolve club guild id: %w", err)
	}

	entitlements, err := s.guildRepo.ResolveEntitlements(ctx, db, guildID)
	if err != nil {
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("resolve betting entitlements: %w", err)
	}

	feature, ok := entitlements.Feature(guildtypes.ClubFeatureBetting)
	if !ok {
		feature = guildtypes.ClubFeatureAccess{
			Key:    guildtypes.ClubFeatureBetting,
			State:  guildtypes.FeatureAccessStateDisabled,
			Source: guildtypes.FeatureAccessSourceNone,
		}
	}

	return guildID, feature, nil
}

// resolveAdminAccess extends resolveAccess with an admin role check.
func (s *BettingService) resolveAdminAccess(
	ctx context.Context,
	db bun.IDB,
	clubUUID, userUUID uuid.UUID,
) (sharedtypes.GuildID, guildtypes.ClubFeatureAccess, error) {
	guildID, access, err := s.resolveAccess(ctx, db, clubUUID, userUUID)
	if err != nil {
		return "", guildtypes.ClubFeatureAccess{}, err
	}
	// Only FeatureAccessStateDisabled fully blocks admin operations.
	// FeatureAccessStateFrozen is intentionally allowed: admins must still be
	// able to perform wallet adjustments and manage historical data (e.g., void
	// markets, force-settle tickets) while the club is in read-only freeze.
	// This mirrors the settle.go invariant: settlement/admin continues during freeze.
	if access.State == guildtypes.FeatureAccessStateDisabled {
		return "", guildtypes.ClubFeatureAccess{}, ErrFeatureDisabled
	}

	membership, err := s.userRepo.GetClubMembership(ctx, db, userUUID, clubUUID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return "", guildtypes.ClubFeatureAccess{}, ErrAdminRequired
		}
		return "", guildtypes.ClubFeatureAccess{}, fmt.Errorf("load admin membership: %w", err)
	}
	if membership.Role != sharedtypes.UserRoleAdmin {
		return "", guildtypes.ClubFeatureAccess{}, ErrAdminRequired
	}

	return guildID, access, nil
}

// resolveRoundFromAdminResettle is a convenience helper for admin resettle flows
// that need to load and validate the round referenced by a market.
func (s *BettingService) resolveRoundFromAdminResettle(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	market *bettingdb.Market,
) (*BettingSettlementRound, error) {
	round, err := s.roundRepo.GetRound(ctx, db, guildID, sharedtypes.RoundID(market.RoundID))
	if err != nil {
		return nil, fmt.Errorf("load round for resettle: %w", err)
	}
	if round == nil || (!bool(round.Finalized) && round.State != roundtypes.RoundStateFinalized) {
		return nil, ErrRoundNotFinalized
	}
	return settlementRoundFromRound(round), nil
}
