package leaderboardservice

import (
	"context"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
)

// BatchTagAssignmentRequested handles multiple manual tag assignments in one operation
func (s *LeaderboardService) BatchTagAssignmentRequested(ctx context.Context, payload leaderboardevents.BatchTagAssignmentRequestedPayload) (LeaderboardOperationResult, error) {
	// Log the batch operation - update to use UUID string representation
	s.logger.InfoContext(ctx, "Batch tag assignment triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("batch_id", payload.BatchID),
		attr.String("requesting_user", string(payload.RequestingUserID)),
		attr.Int("assignment_count", len(payload.Assignments)),
	)

	// Prepare assignments for the database
	dbAssignments := make([]leaderboarddb.TagAssignment, 0, len(payload.Assignments))
	for _, assignment := range payload.Assignments {
		// Validate each tag number (optional)
		if assignment.TagNumber < 0 {
			s.logger.Warn("Skipping invalid tag number",
				attr.String("user_id", string(assignment.UserID)),
				attr.Int("tag_number", int(assignment.TagNumber)),
			)
			continue
		}

		dbAssignments = append(dbAssignments, leaderboarddb.TagAssignment{
			UserID:    sharedtypes.DiscordID(assignment.UserID),
			TagNumber: sharedtypes.TagNumber(assignment.TagNumber),
		})
	}

	return s.serviceWrapper(ctx, "BatchTagAssignmentRequested", func() (LeaderboardOperationResult, error) {
		// Execute batch assignment in a single database operation
		startTime := time.Now()
		err := s.LeaderboardDB.BatchAssignTags(ctx, dbAssignments, leaderboarddb.ServiceUpdateSourceAdminBatch, sharedtypes.RoundID(uuid.Nil), payload.RequestingUserID)
		s.metrics.RecordOperationDuration(ctx, "BatchAssignTags", "BatchTagAssignmentRequested", time.Since(startTime).Seconds())

		if err != nil {
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: payload.RequestingUserID,
					BatchID:          payload.BatchID,
					Reason:           err.Error(),
				},
			}, err
		}

		// Success case
		return LeaderboardOperationResult{
			Success: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: payload.RequestingUserID,
				BatchID:          payload.BatchID,
				AssignmentCount:  len(dbAssignments),
				Assignments:      payload.Assignments,
			},
		}, nil
	})
}
