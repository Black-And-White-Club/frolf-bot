package leaderboardservices

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	"github.com/nats-io/nats.go"
)

// SwapGroup represents a group of users wanting to swap tags.
type SwapGroup struct {
	RequestorID  string
	RequestorTag int
	TargetTag    string
	Requests     []*leaderboardcommands.TagSwapRequest
}

// checkForMatch checks if a swap group contains a matching pair of requests.
func (sg *SwapGroup) checkForMatch() bool {
	if len(sg.Requests) < 2 {
		return false
	}

	// Create a map to store requests by tag numbers
	requestsMap := make(map[string]map[int]map[string]bool)

	for _, request := range sg.Requests {
		if requestsMap[request.RequestorID] == nil {
			requestsMap[request.RequestorID] = make(map[int]map[string]bool)
		}
		if requestsMap[request.RequestorID][request.RequestorTag] == nil {
			requestsMap[request.RequestorID][request.RequestorTag] = make(map[string]bool)
		}
		requestsMap[request.RequestorID][request.RequestorTag][request.TargetTag] = true
	}

	// Check for matching pairs
	for requestorID, requestorTags := range requestsMap {
		for requestorTag, targetTags := range requestorTags {
			for targetTag := range targetTags {
				if requestsMap[targetTag] != nil &&
					requestsMap[targetTag][requestorTag] != nil &&
					requestsMap[targetTag][requestorTag][requestorID] {
					return true // Match found!
				}
			}
		}
	}

	return false // No match found
}

// RemoveSwapGroup removes a swap group from Jetstream.
func (s *LeaderboardService) RemoveSwapGroup(ctx context.Context, js nats.JetStreamContext, swapGroup *SwapGroup) error {
	swapGroupID := fmt.Sprintf("%s-%d-%s", swapGroup.RequestorID, swapGroup.RequestorTag, swapGroup.TargetTag)

	// Delete the swap group data from Jetstream.
	kv, err := js.KeyValue(swapGroupID)
	if err != nil {
		return fmt.Errorf("failed to get key-value store: %w", err)
	}

	if err := kv.Delete(swapGroupID); err != nil { // Removed ctx argument
		return fmt.Errorf("failed to remove swap group from Jetstream: %w", err)
	}

	return nil
}

// StoreSwapGroup stores a swap group in Jetstream.
func (s *LeaderboardService) StoreSwapGroup(ctx context.Context, js nats.JetStreamContext, swapGroup *SwapGroup) error {
	swapGroupID := fmt.Sprintf("%s-%d-%s", swapGroup.RequestorID, swapGroup.RequestorTag, swapGroup.TargetTag)

	// Marshal the swap group data.
	data, err := json.Marshal(swapGroup)
	if err != nil {
		return fmt.Errorf("failed to marshal swap group data: %w", err)
	}

	// Store the swap group data in Jetstream.
	kv, err := js.KeyValue(swapGroupID)
	if err != nil {
		return fmt.Errorf("failed to get key-value store: %w", err)
	}

	_, err = kv.Put(swapGroupID, data) // Removed ctx argument
	if err != nil {
		return fmt.Errorf("failed to store swap group in Jetstream: %w", err)
	}

	return nil
}
