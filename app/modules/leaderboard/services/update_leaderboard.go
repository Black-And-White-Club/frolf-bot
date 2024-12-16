package leaderboardservices

import (
	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
)

// LeaderboardUpdateService handles updating the leaderboard data.
type LeaderboardUpdateService struct {
	// No dependencies needed for now
}

// NewLeaderboardUpdateService creates a new LeaderboardUpdateService.
func NewLeaderboardUpdateService() *LeaderboardUpdateService {
	return &LeaderboardUpdateService{}
}

// UpdateLeaderboardData updates the leaderboard data with the new tag assignments.
func (s *LeaderboardUpdateService) UpdateLeaderboardData(currentLeaderboardData map[int]string, newEntries []leaderboarddto.LeaderboardEntry) map[int]string {
	updatedData := make(map[int]string)
	for _, entry := range newEntries {
		updatedData[entry.TagNumber] = entry.DiscordID
	}
	return updatedData
}
