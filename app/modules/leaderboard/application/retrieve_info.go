package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// GetLeaderboard returns the active leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) (LeaderboardOperationResult, error) {
	// Record leaderboard retrieval attempt
	s.metrics.RecordLeaderboardGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Leaderboard retrieval triggered",
		attr.ExtractCorrelationID(ctx))

	return s.serviceWrapper(ctx, "GetLeaderboard", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// 1. Get the active leaderboard from the database.
		dbStartTime := time.Now()
		leaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordLeaderboardGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get active leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err))

			s.metrics.RecordLeaderboardGetFailure(ctx, "LeaderboardService")

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.GetLeaderboardFailedPayload{
					Reason: "failed to get active leaderboard",
				},
				Error: err,
			}, err
		}

		// 2. Prepare the response payload.
		leaderboardEntries := make([]leaderboardtypes.LeaderboardEntry, 0, len(leaderboard.LeaderboardData))
		for _, entry := range leaderboard.LeaderboardData {
			// Create a pointer to sharedtypes.TagNumber
			tagNumPtr := entry.TagNumber
			leaderboardEntries = append(leaderboardEntries, leaderboardtypes.LeaderboardEntry{
				TagNumber: tagNumPtr,
				UserID:    entry.UserID,
			})
		}

		s.logger.InfoContext(ctx, "Successfully retrieved leaderboard",
			attr.ExtractCorrelationID(ctx))

		s.metrics.RecordLeaderboardGetSuccess(ctx, "LeaderboardService")

		return LeaderboardOperationResult{
			Success: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: leaderboardEntries,
			},
		}, nil
	})
}

// GetTagByUserID returns the tag number for a given user ID.
func (s *LeaderboardService) RoundGetTagByUserID(ctx context.Context, payload sharedevents.RoundTagLookupRequestPayload) (LeaderboardOperationResult, error) {
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Tag retrieval triggered for user",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.RoundID("round_id", payload.RoundID),
		attr.String("original_response", string(payload.Response)),
		attr.Any("original_joined_late", payload.JoinedLate),
	)

	// CORRECTED: Removed payload.RoundID argument from serviceWrapper call
	return s.serviceWrapper(ctx, "RoundGetTagByUserID", func(ctx context.Context) (LeaderboardOperationResult, error) {
		dbStartTime := time.Now()
		tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, payload.UserID)
		s.metrics.RecordOperationDuration(ctx, "GetTagByUserID", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		// Prepare the base result payload, echoing back original context
		resultPayload := sharedevents.RoundTagLookupResultPayload{
			UserID:             payload.UserID,
			RoundID:            payload.RoundID,
			OriginalResponse:   payload.Response,
			OriginalJoinedLate: payload.JoinedLate,
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
				s.logger.InfoContext(ctx, "No tag found for user",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(payload.UserID)))
				s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

				resultPayload.TagNumber = nil
				resultPayload.Found = false
				resultPayload.Error = ""

				return LeaderboardOperationResult{Success: &resultPayload}, nil
			}

			s.logger.ErrorContext(ctx, "Failed to get tag by UserID",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

			resultPayload.TagNumber = nil
			resultPayload.Found = false
			resultPayload.Error = fmt.Sprintf("failed to get tag: %v", err)

			return LeaderboardOperationResult{Success: &resultPayload, Error: err}, fmt.Errorf("failed to get tag by UserID: %w", err)
		}

		s.logger.InfoContext(ctx, "Retrieved tag number",
			attr.ExtractCorrelationID(ctx),
			attr.String("tag_number", fmt.Sprintf("%v", tagNumber)))

		s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

		var tagPtr *sharedtypes.TagNumber
		if tagNumber != nil {
			tagValue := sharedtypes.TagNumber(*tagNumber)
			tagPtr = &tagValue
		} else {
			s.logger.WarnContext(ctx, "LeaderboardDB.GetTagByUserID returned nil tag with no error",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
			)
			resultPayload.TagNumber = nil
			resultPayload.Found = false
			return LeaderboardOperationResult{Success: &resultPayload}, nil
		}

		resultPayload.TagNumber = tagPtr
		resultPayload.Found = true
		resultPayload.Error = ""

		return LeaderboardOperationResult{Success: &resultPayload}, nil
	})
}

// GetTagByUserID returns the tag number for a given user ID.
func (s *LeaderboardService) GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (LeaderboardOperationResult, error) {
	// Record tag retrieval attempt
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Tag retrieval triggered for user",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(userID)))

	return s.serviceWrapper(ctx, "GetTagByUserID", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// Fetch tag number from DB
		dbStartTime := time.Now()
		tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, userID)
		s.metrics.RecordOperationDuration(ctx, "GetTagByUserID", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		if err != nil {
			// Check if this is a "not found" error
			if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
				// This is not an error case, just no tag found for this user
				s.logger.InfoContext(ctx, "No tag found for user",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(userID)))

				s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

				return LeaderboardOperationResult{
					Success: &leaderboardevents.GetTagNumberResponsePayload{
						TagNumber: nil,
						UserID:    userID,
						RoundID:   roundID,
						Found:     false,
					},
				}, nil
			}

			// This is a real error
			s.logger.ErrorContext(ctx, "Failed to get tag by UserID",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err))

			s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.GetTagNumberFailedPayload{
					Reason: "failed to get tag by UserID: " + err.Error(),
				},
				Error: err,
			}, err
		}

		// Properly handle `nil` case
		var tagPtr *sharedtypes.TagNumber
		if tagNumber != nil {
			tagValue := sharedtypes.TagNumber(*tagNumber)
			tagPtr = &tagValue
		}

		s.logger.InfoContext(ctx, "Retrieved tag number",
			attr.ExtractCorrelationID(ctx),
			attr.String("tag_number", fmt.Sprintf("%v", tagPtr)))

		// Return response with correct tag number handling
		s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

		return LeaderboardOperationResult{
			Success: &leaderboardevents.GetTagNumberResponsePayload{
				TagNumber: tagPtr,
				UserID:    userID,
				RoundID:   roundID,
				Found:     tagPtr != nil,
			},
		}, nil
	})
}
