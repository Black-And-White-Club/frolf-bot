package leaderboardservice

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetLeaderboard returns the active leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, msg *message.Message) (LeaderboardOperationResult, error) {
	// Record leaderboard retrieval attempt
	s.metrics.RecordLeaderboardGetAttempt("LeaderboardService")

	s.logger.Info("Leaderboard retrieval triggered",
		attr.CorrelationIDFromMsg(msg))

	return s.serviceWrapper(msg, "GetLeaderboard", func() (LeaderboardOperationResult, error) {
		ctx, span := s.tracer.StartSpan(ctx, "GetLeaderboard.DatabaseOperation", msg)
		defer span.End()

		// 1. Get the active leaderboard from the database.
		dbStartTime := time.Now()
		leaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration("GetActiveLeaderboard", "LeaderboardService", time.Since(dbStartTime).Seconds())
		s.metrics.RecordLeaderboardGetDuration("LeaderboardService", time.Since(dbStartTime).Seconds())
		if err != nil {
			s.logger.Error("Failed to get active leaderboard",
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err))

			s.metrics.RecordLeaderboardGetFailure("LeaderboardService")

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.GetLeaderboardFailedPayload{
					Reason: "failed to get active leaderboard",
				},
				Error: err,
			}, err
		}

		// 2. Prepare the response payload.
		leaderboardEntries := make([]leaderboardevents.LeaderboardEntry, 0, len(leaderboard.LeaderboardData))
		for _, entry := range leaderboard.LeaderboardData {
			// Create a pointer to sharedtypes.TagNumber
			tagNumPtr := entry.TagNumber // Use the TagNumber from the entry
			leaderboardEntries = append(leaderboardEntries, leaderboardevents.LeaderboardEntry{
				TagNumber: &tagNumPtr, // Take the address of the TagNumber
				UserID:    entry.UserID,
			})
		}

		s.logger.Info("Successfully retrieved leaderboard",
			attr.CorrelationIDFromMsg(msg))

		s.metrics.RecordLeaderboardGetSuccess("LeaderboardService")

		return LeaderboardOperationResult{
			Success: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: leaderboardEntries,
			},
		}, nil
	})
}

// GetTagByUserID returns the tag number for a given user ID.
func (s *LeaderboardService) GetTagByUserID(ctx context.Context, msg *message.Message, userID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (LeaderboardOperationResult, error) {
	// Record tag retrieval attempt
	s.metrics.RecordTagGetAttempt("LeaderboardService")

	s.logger.Info("Tag retrieval triggered for user",
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)))

	return s.serviceWrapper(msg, "GetTagByUserID", func() (LeaderboardOperationResult, error) {
		ctx, span := s.tracer.StartSpan(ctx, "GetTagByUserID.DatabaseOperation", msg)
		defer span.End()

		// Fetch tag number from DB
		dbStartTime := time.Now()
		tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, userID)
		s.metrics.RecordOperationDuration("GetTagByUserID", "LeaderboardService", time.Since(dbStartTime).Seconds())
		s.metrics.RecordTagGetDuration("LeaderboardService", time.Since(dbStartTime).Seconds())
		if err != nil {
			s.logger.Error("Failed to get tag by UserID",
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err))

			s.metrics.RecordTagGetFailure("LeaderboardService")

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.GetTagNumberFailedPayload{
					Reason: "failed to get tag by UserID",
				},
				Error: err,
			}, err
		}

		// Properly handle `nil` case
		var tagPtr *sharedtypes.TagNumber
		if tagNumber != nil {
			tagValue := sharedtypes.TagNumber(*tagNumber) // Convert to sharedtypes.TagNumber
			tagPtr = &tagValue                            // Create a new pointer
		}

		s.logger.Info("Retrieved tag number",
			attr.CorrelationIDFromMsg(msg),
			attr.String("tag_number", fmt.Sprintf("%v", tagPtr)))

		// Return response with correct tag number handling
		s.metrics.RecordTagGetSuccess("LeaderboardService")

		return LeaderboardOperationResult{
			Success: &leaderboardevents.GetTagNumberResponsePayload{
				TagNumber: tagPtr,
				UserID:    userID,
				RoundID:   roundID,
			},
		}, nil
	})
}
