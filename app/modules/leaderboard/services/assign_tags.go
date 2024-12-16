package leaderboardservices

import (
	"context"
	"sort"

	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
)

// AssignTags assigns tags to the given leaderboard entries.
func (s *LeaderboardService) AssignTags(ctx context.Context, entries []leaderboarddto.LeaderboardEntry) ([]leaderboarddto.LeaderboardEntry, error) {
	// 1. Sort the entries by TagNumber
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TagNumber < entries[j].TagNumber
	})

	// 2. Assign new tag numbers based on sorted order
	for i := range entries {
		entries[i].TagNumber = i + 1
	}

	return entries, nil
}
