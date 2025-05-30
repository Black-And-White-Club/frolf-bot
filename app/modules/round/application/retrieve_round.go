package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// GetRound retrieves the round from the database.
func (s *RoundService) GetRound(ctx context.Context, roundID sharedtypes.RoundID) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "GetRound", roundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Getting round from database",
			attr.RoundID("round_id", roundID),
		)

		dbRound, err := s.RoundDB.GetRound(ctx, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to retrieve round",
				attr.RoundID("round_id", roundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{ // Add pointer here
					RoundID: roundID,
					Error:   err.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
		}

		s.logger.InfoContext(ctx, "Round retrieved from database",
			attr.RoundID("round_id", roundID),
		)

		rtRound := &roundtypes.Round{ // Make this a pointer for consistency
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
		}

		s.logger.InfoContext(ctx, "Round converted to roundtypes.Round",
			attr.RoundID("round_id", roundID),
		)

		return RoundOperationResult{
			Success: rtRound,
		}, nil
	})
}
