package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GetLeaderboard returns an OperationResult representing the current leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error) {
	s.metrics.RecordLeaderboardGetAttempt(ctx, "LeaderboardService")

	return s.withTelemetry(ctx, "GetLeaderboard", guildID, func(ctx context.Context) (results.OperationResult, error) {
		dbStartTime := time.Now()
		leaderboard, err := s.repo.GetActiveLeaderboard(ctx, guildID)
		s.metrics.RecordLeaderboardGetDuration(ctx, "LeaderboardService", time.Since(dbStartTime))

		if err != nil {
			s.metrics.RecordLeaderboardGetFailure(ctx, "LeaderboardService")
			return results.FailureResult(&leaderboardevents.GetLeaderboardFailedPayloadV1{
				GuildID: guildID,
				Reason:  "database error",
			}), err
		}

		s.metrics.RecordLeaderboardGetSuccess(ctx, "LeaderboardService")

		entries := make([]leaderboardtypes.LeaderboardEntry, len(leaderboard.LeaderboardData))
		copy(entries, leaderboard.LeaderboardData)

		return results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{
			GuildID:     guildID,
			Leaderboard: entries,
		}), nil
	})
}

// RoundGetTagByUserID returns a round-scoped tag lookup result payload.
func (s *LeaderboardService) RoundGetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, payload sharedevents.RoundTagLookupRequestedPayloadV1) (results.OperationResult, error) {
	tag, err := s.GetTagByUserID(ctx, guildID, payload.UserID)
	if err != nil {
		// Not found -> return result payload with Found=false as a failure outcome
		return results.FailureResult(&sharedevents.RoundTagLookupResultPayloadV1{
			ScopedGuildID: sharedevents.ScopedGuildID{GuildID: guildID},
			UserID:        payload.UserID,
			RoundID:       payload.RoundID,
			TagNumber:     nil,
			Found:         false,
		}), nil
	}

	return results.SuccessResult(&sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: guildID},
		UserID:        payload.UserID,
		RoundID:       payload.RoundID,
		TagNumber:     &tag,
		Found:         true,
	}), nil
}

// GetTagByUserID returns the tag number for a user or an error.
func (s *LeaderboardService) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
	s.metrics.RecordTagGetAttempt(ctx, "LeaderboardService")

	dbStartTime := time.Now()
	tagNumber, err := s.repo.GetTagByUserID(ctx, guildID, userID)
	s.metrics.RecordTagGetDuration(ctx, "LeaderboardService", time.Since(dbStartTime))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			return 0, err
		}
		return 0, fmt.Errorf("system error retrieving tag: %w", err)
	}

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

	result, err := s.repo.CheckTagAvailability(ctx, guildID, userID, *tagNumber)
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
