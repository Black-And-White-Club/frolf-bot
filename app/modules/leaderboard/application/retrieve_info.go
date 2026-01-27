package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// TagAvailabilityResult represents the detailed result of a tag availability check.
type TagAvailabilityResult struct {
	Available bool
	Reason    string
}

// GetLeaderboard returns the active leaderboard entries as domain types.
func (s *LeaderboardService) GetLeaderboard(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) ([]leaderboardtypes.LeaderboardEntry, error) {

	leaderboard, err := s.repo.GetActiveLeaderboard(ctx, s.db, guildID)
	if err != nil {
		return nil, err
	}
	if leaderboard == nil {
		return nil, leaderboarddb.ErrNoActiveLeaderboard
	}

	// Return a copy of entries
	entries := make([]leaderboardtypes.LeaderboardEntry, len(leaderboard.LeaderboardData))
	copy(entries, leaderboard.LeaderboardData)
	return entries, nil
}

// GetTagByUserID returns the tag number for a given user.
func (s *LeaderboardService) GetTagByUserID(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (sharedtypes.TagNumber, error) {

	leaderboard, err := s.repo.GetActiveLeaderboard(ctx, nil, guildID)
	if err != nil {
		return 0, err
	}

	for _, entry := range leaderboard.LeaderboardData {
		if entry.UserID == userID {
			return entry.TagNumber, nil
		}
	}

	return 0, sql.ErrNoRows
}

// RoundGetTagByUserID wraps GetTagByUserID for telemetry/results but still returns domain type.
func (s *LeaderboardService) RoundGetTagByUserID(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (sharedtypes.TagNumber, error) {
	return s.GetTagByUserID(ctx, guildID, userID)
}

// CheckTagAvailability returns domain result; handler converts it to payload.
func (s *LeaderboardService) CheckTagAvailability(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
) (TagAvailabilityResult, error) {

	leaderboard, err := s.repo.GetActiveLeaderboard(ctx, nil, guildID)
	if err != nil {
		if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			return TagAvailabilityResult{Available: false, Reason: "no active leaderboard"}, nil
		}
		return TagAvailabilityResult{}, err
	}

	available, reason := checkInternalAvailability(leaderboard, userID, tagNumber)
	return TagAvailabilityResult{Available: available, Reason: reason}, nil
}

// Private helper function
func checkInternalAvailability(l *leaderboarddb.Leaderboard, userID sharedtypes.DiscordID, tag sharedtypes.TagNumber) (bool, string) {
	for _, entry := range l.LeaderboardData {
		if entry.TagNumber == tag && entry.UserID != userID {
			return false, "tag is already taken"
		}
	}
	return true, ""
}
