package leaderboardservices

import (
	"context"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
)

// InitiateTagSwap initiates a tag swap between two users.
func (s *LeaderboardService) InitiateTagSwap(ctx context.Context, swapRequest *leaderboardcommands.TagSwapRequest) (*SwapGroupResult, error) {
	// Create a new swap group.
	swapGroup := &SwapGroup{
		RequestorID:  swapRequest.RequestorID,
		RequestorTag: swapRequest.RequestorTag,
		TargetTag:    swapRequest.TargetTag,
		Requests:     []*leaderboardcommands.TagSwapRequest{swapRequest},
	}

	var result *SwapGroupResult

	// Check if there's a match within the group.
	if matchFound := swapGroup.checkForMatch(); matchFound {
		// Prepare the result indicating a match
		result = &SwapGroupResult{
			MatchFound:   true,
			RequestorID:  swapGroup.Requests[0].RequestorID,
			RequestorTag: swapGroup.Requests[0].RequestorTag,
			TargetID:     swapGroup.Requests[1].RequestorID,
			TargetTag:    swapGroup.Requests[1].TargetTag, // Corrected type mismatch
			SwapGroup:    swapGroup,
		}
	} else {
		// Prepare the result indicating no match
		result = &SwapGroupResult{
			MatchFound:   false,
			RequestorID:  swapRequest.RequestorID,
			RequestorTag: swapRequest.RequestorTag,
			TargetTag:    swapRequest.TargetTag, // Corrected type mismatch
			SwapGroup:    swapGroup,
		}
	}
	return result, nil
}

// SwapGroupResult represents the result of a tag swap initiation.
type SwapGroupResult struct {
	MatchFound   bool
	RequestorID  string
	RequestorTag int
	TargetID     string
	TargetTag    string // Corrected type mismatch
	SwapGroup    *SwapGroup
}
