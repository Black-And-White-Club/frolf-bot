package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GetLeaderboard returns the active leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (LeaderboardOperationResult, error) {
	// Record leaderboard retrieval attempt
	s.metrics.RecordLeaderboardGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Leaderboard retrieval triggered",
		attr.ExtractCorrelationID(ctx))

	return s.serviceWrapper(ctx, "GetLeaderboard", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// 1. Get the active leaderboard from the database.
		dbStartTime := time.Now()
		leaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx, guildID)
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordLeaderboardGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get active leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err))

			s.metrics.RecordLeaderboardGetFailure(ctx, "LeaderboardService")

			// If it's specifically the "no active leaderboard" error
			if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
				return LeaderboardOperationResult{
					Failure: &leaderboardevents.GetLeaderboardFailedPayload{
						Reason: err.Error(), // Use the error message directly
					},
				}, nil
			}

			// For other database errors, return both failure and error
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.GetLeaderboardFailedPayload{
					Reason: "Database error when retrieving leaderboard",
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

func (s *LeaderboardService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, payload sharedevents.RoundTagLookupRequestPayload) (LeaderboardOperationResult, error) {
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Tag retrieval triggered for user (Round)",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.RoundID("round_id", payload.RoundID),
		attr.String("original_response", string(payload.Response)),
		attr.Any("original_joined_late", payload.JoinedLate),
	)

	return s.serviceWrapper(ctx, "RoundGetTagByUserID", func(ctx context.Context) (LeaderboardOperationResult, error) {
		dbStartTime := time.Now()
		tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, guildID, payload.UserID)
		s.metrics.RecordOperationDuration(ctx, "GetTagByUserID", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		resultPayload := sharedevents.RoundTagLookupResultPayload{
			UserID:             payload.UserID,
			RoundID:            payload.RoundID,
			OriginalResponse:   payload.Response,
			OriginalJoinedLate: payload.JoinedLate,
		}

		// Handle errors from the database call
		if err != nil {
			// Specific handling for no active leaderboard
			if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
				s.logger.ErrorContext(ctx, "Failed to get tag by UserID: No active leaderboard found (System Error)",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

				return LeaderboardOperationResult{
					Failure: &sharedevents.RoundTagLookupFailedPayload{
						UserID:  payload.UserID,
						RoundID: payload.RoundID,
						Reason:  "No active leaderboard found",
					},
				}, nil // Return nil standard error as this is a handled business error
			}

			// Specific handling for user not found (sql.ErrNoRows)
			if errors.Is(err, sql.ErrNoRows) {
				s.logger.InfoContext(ctx, "No tag found for user in active leaderboard data (Business Outcome - sql.ErrNoRows)",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

				resultPayload.TagNumber = nil
				resultPayload.Found = false
				resultPayload.Error = err.Error() // Set Error field with the actual error string

				return LeaderboardOperationResult{Success: &resultPayload}, nil // Return nil standard error as this is a handled business outcome
			}

			// General handling for any other unexpected database errors
			s.logger.ErrorContext(ctx, "Failed to get tag by UserID due to unexpected DB error (Round)",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

			// Return the error in the result struct and as a standard error
			return LeaderboardOperationResult{Error: fmt.Errorf("failed to get tag by UserID (Round): %w", err)}, fmt.Errorf("failed to get tag by UserID (Round): %w", err)
		}

		// Handle the case where err is nil, but tagNumber is also nil (should be treated as not found)
		if tagNumber == nil {
			s.logger.InfoContext(ctx, "No tag found for user in active leaderboard data (Business Outcome - nil tagNumber and nil error)",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
			)
			s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

			resultPayload.TagNumber = nil
			resultPayload.Found = false
			resultPayload.Error = "" // Error is empty string if nil tagNumber and nil error from DB

			return LeaderboardOperationResult{Success: &resultPayload}, nil // Return nil standard error as this is a handled business outcome
		}

		// Success path: err is nil and tagNumber is non-nil
		s.logger.InfoContext(ctx, "Retrieved tag number for user (Round)",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(payload.UserID)),
			attr.Any("tag_number", *tagNumber), // Log the dereferenced tag number
		)
		s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

		tagValue := sharedtypes.TagNumber(*tagNumber)
		resultPayload.TagNumber = &tagValue
		resultPayload.Found = true
		resultPayload.Error = ""

		return LeaderboardOperationResult{Success: &resultPayload}, nil // Return nil standard error on success
	})
}

// GetTagByUserID returns the tag number for a given user ID.
// This method is used by handlers that need a simple tag lookup result.
// It now accepts a TagNumberRequestPayload struct.
func (s *LeaderboardService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (LeaderboardOperationResult, error) {
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	s.logger.InfoContext(ctx, "Tag retrieval triggered for user",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(userID)),
	)

	return s.serviceWrapper(ctx, "GetTagByUserID", func(ctx context.Context) (LeaderboardOperationResult, error) {
		dbStartTime := time.Now()
		tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, guildID, userID)
		s.metrics.RecordOperationDuration(ctx, "GetTagByUserID", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		if err != nil {
			if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
				s.logger.ErrorContext(ctx, "Failed to get tag by UserID: No active leaderboard found (System Error)",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)
				s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

				// Use the same pattern as RoundGetTagByUserID, returning a failure payload
				return LeaderboardOperationResult{
					Failure: &sharedevents.DiscordTagLookupByUserIDFailedPayload{
						UserID: userID,
						Reason: "No active leaderboard found",
					},
				}, nil
			}

			if errors.Is(err, sql.ErrNoRows) {
				s.logger.InfoContext(ctx, "No tag found for user",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(userID)),
				)

				s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

				return LeaderboardOperationResult{
					Success: &sharedevents.DiscordTagLookupResultPayload{
						TagNumber: nil,
						UserID:    userID,
						Found:     false,
					},
				}, nil
			}

			s.logger.ErrorContext(ctx, "Failed to get tag by UserID due to unexpected DB error",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)

			s.metrics.RecordTagGetFailure(ctx, "LeaderboardService")

			return LeaderboardOperationResult{
				Error: fmt.Errorf("failed to get tag by UserID: %w", err),
			}, nil
		}

		var tagPtr *sharedtypes.TagNumber
		if tagNumber != nil {
			tagValue := sharedtypes.TagNumber(*tagNumber)
			tagPtr = &tagValue
		}

		s.logger.InfoContext(ctx, "Retrieved tag number",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(userID)),
			attr.String("tag_number", fmt.Sprintf("%v", tagPtr)))

		s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

		return LeaderboardOperationResult{
			Success: &sharedevents.DiscordTagLookupResultPayload{
				TagNumber: tagPtr,
				UserID:    userID,
				Found:     tagPtr != nil,
			},
		}, nil
	})
}
