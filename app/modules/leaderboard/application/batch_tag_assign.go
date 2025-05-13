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
	s.logger.InfoContext(ctx, "Batch tag assignment triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("batch_id", payload.BatchID),
		attr.String("requesting_user", string(payload.RequestingUserID)),
		attr.Int("assignment_count", len(payload.Assignments)),
	)

	// Prepare assignments for the database, skipping invalid ones
	dbAssignments := make([]leaderboarddb.TagAssignment, 0, len(payload.Assignments))
	processedAssignmentsInfo := make([]leaderboardevents.TagAssignmentInfo, 0, len(payload.Assignments))
	for _, assignment := range payload.Assignments {
		if assignment.TagNumber < 0 {
			s.logger.Warn("Skipping invalid tag number",
				attr.String("user_id", string(assignment.UserID)),
				attr.Int("tag_number", int(assignment.TagNumber)),
			)
			continue
		}

		dbAssignments = append(dbAssignments, leaderboarddb.TagAssignment{
			UserID:    assignment.UserID,
			TagNumber: assignment.TagNumber,
		})
		// Add to the list for the success payload
		processedAssignmentsInfo = append(processedAssignmentsInfo, assignment)
	}

	// If no valid assignments, return success with count 0
	if len(dbAssignments) == 0 {
		s.logger.InfoContext(ctx, "No valid assignments in batch, completing with success",
			attr.ExtractCorrelationID(ctx),
			attr.String("batch_id", payload.BatchID),
		)
		return LeaderboardOperationResult{
			Success: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: payload.RequestingUserID,
				BatchID:          payload.BatchID,
				AssignmentCount:  0,
				Assignments:      []leaderboardevents.TagAssignmentInfo{}, // Empty slice
			},
		}, nil
	}

	return s.serviceWrapper(ctx, "BatchTagAssignmentRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {
		startTime := time.Now()
		// Pass dbAssignments to the repository
		err := s.LeaderboardDB.BatchAssignTags(ctx, dbAssignments, leaderboarddb.ServiceUpdateSourceAdminBatch, sharedtypes.RoundID(uuid.Nil), payload.RequestingUserID)
		s.metrics.RecordOperationDuration(ctx, "BatchAssignTags", "BatchTagAssignmentRequested", time.Duration(time.Since(startTime).Seconds()))

		if err != nil {
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: payload.RequestingUserID,
					BatchID:          payload.BatchID,
					Reason:           err.Error(),
				},
			}, err
		}

		// Success case - return the processed assignments
		return LeaderboardOperationResult{
			Success: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: payload.RequestingUserID,
				BatchID:          payload.BatchID,
				AssignmentCount:  len(processedAssignmentsInfo),
				Assignments:      processedAssignmentsInfo,
			},
		}, nil
	})
}
