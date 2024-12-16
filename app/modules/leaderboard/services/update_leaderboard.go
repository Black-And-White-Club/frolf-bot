package leaderboardservices

import (
	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
)

// UpdateLeaderboardData updates the leaderboard data with the new tag assignments.
func (s *LeaderboardService) UpdateLeaderboardData(currentLeaderboardData map[int]string, newEntries []leaderboarddto.LeaderboardEntry) map[int]string {
	updatedData := make(map[int]string)
	for _, entry := range newEntries {
		updatedData[entry.TagNumber] = entry.DiscordID
	}
	return updatedData
}
