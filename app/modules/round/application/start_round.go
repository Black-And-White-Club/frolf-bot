package roundservice

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
)

// StartRound handles the start of a round, updates participant data, updates DB, and notifies Discord.
// Multi-guild: require guildID for all round operations
func (s *RoundService) StartRound(
	ctx context.Context,
	req *roundtypes.StartRoundRequest,
) (StartRoundResult, error) {
	guildID := req.GuildID
	roundID := req.RoundID

	startOp := func(ctx context.Context, db bun.IDB) (results.OperationResult[*roundtypes.Round, error], error) {
		s.logger.InfoContext(ctx, "Processing round start",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		// Fetch the round from DB (DB is the source of truth)
		round, err := s.repo.GetRound(ctx, db, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round from database",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return results.FailureResult[*roundtypes.Round, error](err), nil
		}

		// Ensure we have an event message id to update/notify Discord
		if round.EventMessageID == "" {
			return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("round missing event_message_id")), nil
		}

		// Update the round state to "in progress"
		err = s.repo.UpdateRoundState(ctx, db, guildID, roundID, roundtypes.RoundStateInProgress)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round state to in progress",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRoundState")
			return results.FailureResult[*roundtypes.Round, error](err), nil
		}

		// Update local object state to reflect DB change
		round.State = roundtypes.RoundStateInProgress

		return results.SuccessResult[*roundtypes.Round, error](round), nil
	}

	return withTelemetry[*roundtypes.Round, error](s, ctx, "StartRound", roundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		return runInTx[*roundtypes.Round, error](s, ctx, startOp)
	})
}
