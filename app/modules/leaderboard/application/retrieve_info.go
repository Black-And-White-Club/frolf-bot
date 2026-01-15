package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GetLeaderboard matches the Interface: returns (LeaderboardOperationResult, error)
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (LeaderboardOperationResult, error) {
	s.metrics.RecordLeaderboardGetAttempt(ctx, "LeaderboardService")

	return s.serviceWrapper(ctx, "GetLeaderboard", func(ctx context.Context) (LeaderboardOperationResult, error) {
		dbStartTime := time.Now()
		leaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx, guildID)
		s.metrics.RecordLeaderboardGetDuration(ctx, "LeaderboardService", time.Since(dbStartTime))

		if err != nil {
			// For any error from the DB surface surface a friendly failure reason while preserving the
			// underlying error for debugging. Tests expect a specific failure reason string.
			s.metrics.RecordLeaderboardGetFailure(ctx, "LeaderboardService")
			return LeaderboardOperationResult{Err: fmt.Errorf("Database error when retrieving leaderboard: %w", err)}, nil
		}

		s.metrics.RecordLeaderboardGetSuccess(ctx, "LeaderboardService")
		return LeaderboardOperationResult{
			Leaderboard: leaderboard.LeaderboardData,
		}, nil
	})
}

// RoundGetTagByUserID matches the Interface: returns (LeaderboardOperationResult, error)
func (s *LeaderboardService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, payload sharedevents.RoundTagLookupRequestedPayloadV1) (LeaderboardOperationResult, error) {
	// We call GetTagByUserID (which now returns a raw tag) and wrap it for the event bus
	tag, err := s.GetTagByUserID(ctx, guildID, payload.UserID)
	if err != nil {
		// If not found or error, return empty data per typical round flow expectations
		return LeaderboardOperationResult{Leaderboard: leaderboardtypes.LeaderboardData{}}, nil
	}

	return LeaderboardOperationResult{
		Leaderboard: leaderboardtypes.LeaderboardData{
			{
				UserID:    payload.UserID,
				TagNumber: tag,
			},
		},
	}, nil
}

// GetTagByUserID matches the Interface: returns (sharedtypes.TagNumber, error)
// Note: Removed serviceWrapper here because serviceWrapper expects LeaderboardOperationResult
func (s *LeaderboardService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	dbStartTime := time.Now()
	tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, guildID, userID)
	s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Since(dbStartTime))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			return 0, err
		}
		return 0, fmt.Errorf("system error retrieving tag: %w", err)
	}

	// Some repository implementations (or mocks) may return (nil, nil) to indicate
	// that no tag exists for the user. Treat that case as sql.ErrNoRows so callers
	// (like RoundGetTagByUserID) can handle it as "not found".
	if tagNumber == nil {
		return 0, sql.ErrNoRows
	}

	s.metrics.RecordTagGetSuccess(ctx, "LeaderboardService")

	return sharedtypes.TagNumber(*tagNumber), nil
}

// CheckTagAvailability validates whether a tag can be assigned to a user.
func (s *LeaderboardService) CheckTagAvailability(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	tagNumber *sharedtypes.TagNumber,
) (sharedevents.TagAvailabilityCheckResultPayloadV1, *sharedevents.TagAvailabilityCheckFailedPayloadV1, error) {
	if tagNumber == nil {
		failure := &sharedevents.TagAvailabilityCheckFailedPayloadV1{
			GuildID:   guildID,
			UserID:    userID,
			TagNumber: tagNumber,
			Reason:    "tag number is required",
		}
		return sharedevents.TagAvailabilityCheckResultPayloadV1{}, failure, nil
	}

	result, err := s.LeaderboardDB.CheckTagAvailability(ctx, guildID, userID, *tagNumber)
	if err != nil {
		return sharedevents.TagAvailabilityCheckResultPayloadV1{}, nil, err
	}

	return sharedevents.TagAvailabilityCheckResultPayloadV1{
		GuildID:   guildID,
		UserID:    userID,
		TagNumber: tagNumber,
		Available: result.Available,
		Reason:    result.Reason,
	}, nil, nil
}
