package bettingservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// SuspendOpenMarketsForClub suspends all currently open betting markets for a
// club in response to a feature-access change (freeze or disable).
//
// Invariants:
//   - Only open markets are suspended; locked/settled/voided markets are unchanged.
//   - Accepted tickets on suspended markets remain valid and will settle normally
//     when the round is finalized.
//   - If the guild is not found or has no open markets, the call is a no-op.
func (s *BettingService) SuspendOpenMarketsForClub(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) ([]MarketSuspendedResult, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "SuspendOpenMarketsForClub", "betting")

	run := func(ctx context.Context, db bun.IDB) ([]MarketSuspendedResult, error) {
		clubUUID, err := s.userRepo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve club uuid by guild id: %w", err)
		}

		refs, err := s.repo.SuspendOpenMarketsForClub(ctx, db, clubUUID)
		if err != nil {
			return nil, fmt.Errorf("suspend open markets for club %s: %w", clubUUID, err)
		}

		results := make([]MarketSuspendedResult, 0, len(refs))
		for _, ref := range refs {
			results = append(results, MarketSuspendedResult{
				GuildID:  guildID,
				ClubUUID: clubUUID.String(),
				RoundID:  sharedtypes.RoundID(ref.RoundID),
				MarketID: ref.ID,
			})
		}
		return results, nil
	}

	results, err := runInTx(ctx, s.db, nil, run)
	if err != nil {
		s.metrics.RecordOperationFailure(ctx, "SuspendOpenMarketsForClub", "betting")
		s.logError(ctx, "betting.operation.failed", "SuspendOpenMarketsForClub failed", err,
			attr.String("guild_id", string(guildID)),
		)
		return nil, err
	}

	if len(results) > 0 {
		s.logInfo(ctx, "betting.markets.suspended", "open markets suspended due to entitlement loss",
			attr.String("guild_id", string(guildID)),
			attr.Int("count", len(results)),
		)
	}
	s.metrics.RecordOperationSuccess(ctx, "SuspendOpenMarketsForClub", "betting")
	s.metrics.RecordOperationDuration(ctx, "SuspendOpenMarketsForClub", "betting", time.Since(start))
	return results, nil
}
