package leaderboardservice

import (
	"context"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
)

func (s *LeaderboardService) BatchTagAssignmentRequested(ctx context.Context, payload sharedevents.BatchTagAssignmentRequestedPayload) (LeaderboardOperationResult, error) {
	s.logger.InfoContext(ctx, "Batch tag assignment triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("batch_id", payload.BatchID),
		attr.String("requesting_user", string(payload.RequestingUserID)),
		attr.Int("assignment_count", len(payload.Assignments)),
	)

	dbAssignments := make([]leaderboarddb.TagAssignment, 0, len(payload.Assignments))
	processedAssignmentsInfo := make([]sharedevents.TagAssignmentInfo, 0, len(payload.Assignments))
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
		processedAssignmentsInfo = append(processedAssignmentsInfo, assignment)
	}

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
				Assignments:      []leaderboardevents.TagAssignmentInfo{},
			},
		}, nil
	}

	return s.serviceWrapper(ctx, "BatchTagAssignmentRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {
		startTime := time.Now()
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

		convertedAssignments := make([]leaderboardevents.TagAssignmentInfo, len(processedAssignmentsInfo))
		for i, assignment := range processedAssignmentsInfo {
			convertedAssignments[i] = leaderboardevents.TagAssignmentInfo{
				UserID:    assignment.UserID,
				TagNumber: assignment.TagNumber,
			}
		}

		return LeaderboardOperationResult{
			Success: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: payload.RequestingUserID,
				BatchID:          payload.BatchID,
				AssignmentCount:  len(processedAssignmentsInfo),
				Assignments:      convertedAssignments,
			},
		}, nil
	})
}
