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
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *BettingService) UpdateSettings(ctx context.Context, req UpdateSettingsRequest) (*MemberSettings, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "UpdateSettings", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.UpdateSettings")
		defer span.End()
		span.SetAttributes(attribute.String("betting.club_uuid", req.ClubUUID.String()))
	}

	if req.ClubUUID == uuid.Nil || req.UserUUID == uuid.Nil {
		s.metrics.RecordOperationFailure(ctx, "UpdateSettings", "betting")
		return nil, ErrMembershipRequired
	}

	run := func(ctx context.Context, db bun.IDB) (*MemberSettings, error) {
		_, access, err := s.resolveAccess(ctx, db, req.ClubUUID, req.UserUUID)
		if err != nil {
			return nil, err
		}

		switch access.State {
		case guildtypes.FeatureAccessStateDisabled:
			s.metrics.RecordAccessDenied(ctx, "disabled")
			s.metrics.RecordOperationFailure(ctx, "UpdateSettings", "betting")
			s.logWarn(ctx, "betting.settings.access.denied", "UpdateSettings access denied: feature disabled",
				attr.UUIDValue("club_uuid", req.ClubUUID),
			)
			return nil, ErrFeatureDisabled
		case guildtypes.FeatureAccessStateFrozen:
			s.metrics.RecordAccessDenied(ctx, "frozen")
			s.metrics.RecordOperationFailure(ctx, "UpdateSettings", "betting")
			s.logWarn(ctx, "betting.settings.access.denied", "UpdateSettings access denied: feature frozen",
				attr.UUIDValue("club_uuid", req.ClubUUID),
			)
			return nil, ErrFeatureFrozen
		}

		now := time.Now().UTC()
		setting := &bettingdb.MemberSetting{
			ClubUUID:        req.ClubUUID,
			UserUUID:        req.UserUUID,
			OptOutTargeting: req.OptOutTargeting,
			UpdatedAt:       now,
		}
		if err := s.repo.UpsertMemberSettings(ctx, db, setting); err != nil {
			return nil, fmt.Errorf("save betting settings: %w", err)
		}

		return &MemberSettings{
			OptOutTargeting: setting.OptOutTargeting,
			UpdatedAt:       setting.UpdatedAt,
		}, nil
	}

	result, err := runInTx(ctx, s.db, &sql.TxOptions{}, run)
	if err != nil {
		if !errors.Is(err, ErrFeatureDisabled) && !errors.Is(err, ErrFeatureFrozen) {
			s.metrics.RecordOperationFailure(ctx, "UpdateSettings", "betting")
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.RecordError(err)
			}
			s.logError(ctx, "betting.settings.update.failed", "UpdateSettings failed", err,
				attr.UUIDValue("club_uuid", req.ClubUUID),
			)
		}
		return nil, err
	}

	s.metrics.RecordOperationSuccess(ctx, "UpdateSettings", "betting")
	s.metrics.RecordOperationDuration(ctx, "UpdateSettings", "betting", time.Since(start))
	s.logInfo(ctx, "betting.settings.updated", "member settings updated",
		attr.UUIDValue("club_uuid", req.ClubUUID),
		attr.UUIDValue("user_uuid", req.UserUUID),
	)

	return result, nil
}

func (s *BettingService) AdjustWallet(ctx context.Context, req AdjustWalletRequest) (*WalletJournal, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "AdjustWallet", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.AdjustWallet")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.club_uuid", req.ClubUUID.String()),
			attribute.Int("betting.wallet_amount", req.Amount),
		)
	}

	if req.ClubUUID == uuid.Nil || req.AdminUUID == uuid.Nil {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		return nil, ErrAdminRequired
	}
	if req.MemberID == "" {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		return nil, ErrTargetMemberNotFound
	}
	if req.Amount == 0 {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		return nil, ErrAdjustmentAmountInvalid
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		return nil, ErrAdjustmentReasonRequired
	}
	const maxReasonLength = 1000
	if len(reason) > maxReasonLength {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		return nil, ErrReasonTooLong
	}

	direction := "credit"
	if req.Amount < 0 {
		direction = "debit"
	}

	run := func(ctx context.Context, db bun.IDB) (*WalletJournal, error) {
		guildID, _, err := s.resolveAdminAccess(ctx, db, req.ClubUUID, req.AdminUUID)
		if err != nil {
			return nil, err
		}

		targetUserUUID, err := s.userRepo.GetUUIDByDiscordID(ctx, db, req.MemberID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, ErrTargetMemberNotFound
			}
			return nil, fmt.Errorf("resolve target member uuid: %w", err)
		}

		if _, err := s.userRepo.GetClubMembership(ctx, db, targetUserUUID, req.ClubUUID); err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, ErrTargetMemberNotFound
			}
			return nil, fmt.Errorf("load target club membership: %w", err)
		}

		seasonID := defaultSeasonID
		activeSeason, err := s.leaderboardRepo.GetActiveSeason(ctx, db, string(guildID))
		if err != nil {
			return nil, fmt.Errorf("load active season: %w", err)
		}
		if activeSeason != nil && activeSeason.ID != "" {
			seasonID = activeSeason.ID
		}

		entry := &bettingdb.WalletJournalEntry{
			ClubUUID:  req.ClubUUID,
			UserUUID:  targetUserUUID,
			SeasonID:  seasonID,
			EntryType: "admin_adjustment",
			Amount:    req.Amount,
			Reason:    reason,
			CreatedBy: req.AdminUUID.String(),
		}
		if err := s.repo.CreateWalletJournalEntry(ctx, db, entry); err != nil {
			return nil, fmt.Errorf("create betting wallet journal entry: %w", err)
		}

		if err := s.repo.ApplyWalletBalanceDelta(ctx, db, req.ClubUUID, targetUserUUID, seasonID, req.Amount, 0); err != nil {
			return nil, fmt.Errorf("update wallet balance on adjustment: %w", err)
		}

		if err := s.repo.CreateAuditLog(ctx, db, &bettingdb.AuditLog{
			ClubUUID:      req.ClubUUID,
			ActorUserUUID: &req.AdminUUID,
			Action:        "wallet_adjustment",
			Reason:        reason,
			Metadata:      fmt.Sprintf("member_id=%s amount=%d season_id=%s", req.MemberID, req.Amount, seasonID),
		}); err != nil {
			return nil, fmt.Errorf("create betting audit log: %w", err)
		}

		return &WalletJournal{
			ID:        entry.ID,
			EntryType: entry.EntryType,
			Amount:    entry.Amount,
			Reason:    entry.Reason,
			CreatedAt: entry.CreatedAt,
		}, nil
	}

	result, err := runInTx(ctx, s.db, &sql.TxOptions{}, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "AdjustWallet", "betting")
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		s.logError(ctx, "betting.wallet.adjust.failed", "AdjustWallet failed", err,
			attr.UUIDValue("club_uuid", req.ClubUUID),
			attr.String("direction", direction),
			attr.Int("amount", req.Amount),
		)
		return nil, err
	}

	s.metrics.RecordWalletAdjustment(ctx, direction)
	s.metrics.RecordOperationSuccess(ctx, "AdjustWallet", "betting")
	s.metrics.RecordOperationDuration(ctx, "AdjustWallet", "betting", time.Since(start))

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.String("betting.wallet_direction", direction))
	}

	s.logInfo(ctx, "betting.wallet.adjusted", "wallet adjusted",
		attr.UUIDValue("club_uuid", req.ClubUUID),
		attr.String("direction", direction),
		attr.Int("amount", req.Amount),
	)

	return result, nil
}

func (s *BettingService) resolveWallet(
	ctx context.Context,
	db bun.IDB,
	clubUUID, userUUID uuid.UUID,
	guildID sharedtypes.GuildID,
) (resolvedWallet, error) {
	seasonID, seasonName, seasonPoints, err := s.resolveSeasonPoints(ctx, db, userUUID, guildID)
	if err != nil {
		return resolvedWallet{}, err
	}

	bettingBalance, err := s.repo.GetWalletJournalBalance(ctx, db, clubUUID, userUUID, seasonID)
	if err != nil {
		return resolvedWallet{}, fmt.Errorf("load betting wallet balance: %w", err)
	}

	reserved, err := s.repo.GetReservedStakeTotal(ctx, db, clubUUID, userUUID, seasonID)
	if err != nil {
		return resolvedWallet{}, fmt.Errorf("load reserved betting balance: %w", err)
	}

	return resolvedWallet{
		seasonID:       seasonID,
		seasonName:     seasonName,
		seasonPoints:   seasonPoints,
		bettingBalance: bettingBalance,
		reserved:       reserved,
	}, nil
}

// MirrorPointsToWallet journals season-point deltas from a round into the
// betting wallet for each awarded player. Idempotent: a given roundID is
// only journaled once per user per season (enforced by DB unique index).
func (s *BettingService) MirrorPointsToWallet(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	points map[sharedtypes.DiscordID]int,
) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "MirrorPointsToWallet", "betting")

	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "betting.MirrorPointsToWallet")
		defer span.End()
		span.SetAttributes(
			attribute.String("betting.guild_id", string(guildID)),
			attribute.String("betting.round_id", roundID.String()),
		)
	}

	clubUUID, err := s.userRepo.GetClubUUIDByDiscordGuildID(ctx, s.db, guildID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) || clubUUID == uuid.Nil {
			// No betting setup for this guild — silently no-op.
			s.metrics.RecordOperationSuccess(ctx, "MirrorPointsToWallet", "betting")
			return nil
		}
		s.metrics.RecordOperationFailure(ctx, "MirrorPointsToWallet", "betting")
		return fmt.Errorf("resolve club uuid: %w", err)
	}
	if clubUUID == uuid.Nil {
		s.metrics.RecordOperationSuccess(ctx, "MirrorPointsToWallet", "betting")
		return nil
	}

	seasonID := defaultSeasonID
	activeSeason, err := s.leaderboardRepo.GetActiveSeason(ctx, s.db, string(guildID))
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "MirrorPointsToWallet", "betting")
		return fmt.Errorf("load active season: %w", err)
	}
	if activeSeason != nil && activeSeason.ID != "" {
		seasonID = activeSeason.ID
	}

	roundUUID := uuid.UUID(roundID)

	for discordID, pts := range points {
		if pts == 0 {
			continue
		}

		userUUID, err := s.userRepo.GetUUIDByDiscordID(ctx, s.db, discordID)
		if err != nil {
			s.logWarn(ctx, "betting.mirror_points.user_not_found", "MirrorPointsToWallet: could not resolve user, skipping",
				attr.String("discord_id", string(discordID)),
				attr.Error(err),
			)
			continue
		}

		txErr := mirrorTx(ctx, s.db, func(ctx context.Context, tx bun.IDB) error {
			entry := &bettingdb.WalletJournalEntry{
				ClubUUID:      clubUUID,
				UserUUID:      userUUID,
				SeasonID:      seasonID,
				EntryType:     "season_points_awarded",
				Amount:        pts,
				Reason:        "season points awarded",
				CreatedBy:     "system:betting",
				SourceRoundID: &roundUUID,
			}
			if err := s.repo.CreateWalletJournalEntry(ctx, tx, entry); err != nil {
				if strings.Contains(err.Error(), "idx_betting_wallet_journal_dedup") ||
					strings.Contains(err.Error(), "duplicate key") {
					s.logWarn(ctx, "betting.mirror_points.duplicate", "MirrorPointsToWallet: duplicate journal entry, skipping (idempotent replay)",
						attr.String("discord_id", string(discordID)),
						attr.String("round_id", roundUUID.String()),
					)
					return nil
				}
				return fmt.Errorf("create wallet journal entry: %w", err)
			}
			if err := s.repo.ApplyWalletBalanceDelta(ctx, tx, clubUUID, userUUID, seasonID, pts, 0); err != nil {
				return fmt.Errorf("apply wallet balance delta: %w", err)
			}
			return nil
		})
		if txErr != nil {
			s.metrics.RecordOperationFailure(ctx, "MirrorPointsToWallet", "betting")
			s.logError(ctx, "betting.mirror_points.failed", "MirrorPointsToWallet: transaction failed for user", txErr,
				attr.String("discord_id", string(discordID)),
			)
			return txErr
		}
	}

	s.metrics.RecordOperationSuccess(ctx, "MirrorPointsToWallet", "betting")
	s.metrics.RecordOperationDuration(ctx, "MirrorPointsToWallet", "betting", time.Since(start))
	s.logInfo(ctx, "betting.mirror_points.done", "MirrorPointsToWallet complete",
		attr.String("guild_id", string(guildID)),
		attr.String("round_id", roundUUID.String()),
	)

	return nil
}

func (s *BettingService) resolveSeasonPoints(
	ctx context.Context,
	db bun.IDB,
	userUUID uuid.UUID,
	guildID sharedtypes.GuildID,
) (string, string, int, error) {
	seasonID := defaultSeasonID
	seasonName := defaultSeasonName

	activeSeason, err := s.leaderboardRepo.GetActiveSeason(ctx, db, string(guildID))
	if err != nil {
		return "", "", 0, fmt.Errorf("load active season: %w", err)
	}
	if activeSeason != nil {
		seasonID = activeSeason.ID
		seasonName = activeSeason.Name
	}

	user, err := s.userRepo.GetUserByUUID(ctx, db, userUUID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return seasonID, seasonName, 0, nil
		}
		return "", "", 0, fmt.Errorf("load betting user: %w", err)
	}
	if user == nil || user.UserID == nil {
		return seasonID, seasonName, 0, nil
	}

	standing, err := s.leaderboardRepo.GetSeasonStanding(ctx, db, string(guildID), *user.UserID)
	if err != nil {
		return "", "", 0, fmt.Errorf("load betting season standing: %w", err)
	}
	if standing == nil {
		return seasonID, seasonName, 0, nil
	}
	if standing.SeasonID != "" {
		seasonID = standing.SeasonID
	}

	return seasonID, seasonName, standing.TotalPoints, nil
}
