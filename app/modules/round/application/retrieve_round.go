package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// GetRound retrieves the round from the database.
// Multi-guild: require guildID for all round operations
func (s *RoundService) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "GetRound", roundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Getting round from database",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		dbRound, err := s.repo.GetRound(ctx, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to retrieve round",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Error:   err.Error(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Round retrieved from database",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		rtRound := &roundtypes.Round{
			ID:           dbRound.ID,
			Title:        dbRound.Title,
			Description:  dbRound.Description,
			Location:     dbRound.Location,
			EventType:    dbRound.EventType,
			StartTime:    dbRound.StartTime,
			Finalized:    dbRound.Finalized,
			CreatedBy:    dbRound.CreatedBy,
			State:        roundtypes.RoundState(dbRound.State),
			Participants: dbRound.Participants,
			GuildID:      dbRound.GuildID,
		}

		s.logger.InfoContext(ctx, "Round converted to roundtypes.Round",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)

		return results.OperationResult{
			Success: rtRound,
		}, nil
	})
}
