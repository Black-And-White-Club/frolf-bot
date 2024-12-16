// leaderboard_service.go
package leaderboardservices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/google/uuid"
)

// LeaderboardService handles leaderboard-related logic and database interactions.
type LeaderboardService struct {
	db                 leaderboarddb.LeaderboardDB
	natsConnectionPool *nats.NatsConnectionPool
	swapRequests       map[string]map[int]tagSwapRequest
	swapRequestsMu     sync.Mutex
}

// tagSwapRequest represents a request to swap tags.
type tagSwapRequest struct {
	discordID  string
	tagNumber  int
	resultChan chan error
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(db leaderboarddb.LeaderboardDB, natsConnectionPool *nats.NatsConnectionPool) *LeaderboardService {
	return &LeaderboardService{
		db:                 db,
		natsConnectionPool: natsConnectionPool,
		swapRequests:       make(map[string]map[int]tagSwapRequest),
	}
}

// GetLeaderboard retrieves the active leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) (*structs.Leaderboard, error) {
	modelLeaderboard, err := s.db.GetLeaderboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Convert *models.Leaderboard to *structs.Leaderboard
	apiLeaderboard := &structs.Leaderboard{
		ID:              modelLeaderboard.ID,
		LeaderboardData: modelLeaderboard.LeaderboardData,
		Active:          modelLeaderboard.Active,
	}

	return apiLeaderboard, nil
}

// GetTagInfo retrieves the tag information for a user, including availability.
func (s *LeaderboardService) GetTagInfo(ctx context.Context, discordID string, tagNumber int) (bool, *int, error) {
	// 1. Fetch the leaderboard data
	leaderboard, err := s.db.GetLeaderboardTagData(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get leaderboard tag data: %w", err)
	}

	// 2. Marshal the LeaderboardData to JSON
	jsonData, err := json.Marshal(leaderboard.LeaderboardData)
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal leaderboard data to JSON: %w", err)
	}

	// 3. Parse the JSON and check tag availability
	var entries []structs.LeaderboardEntry
	err = json.Unmarshal(jsonData, &entries)
	if err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal leaderboard data from JSON: %w", err)
	}

	var userTagNumber *int
	isTagAvailable := true
	for _, entry := range entries {
		if entry.DiscordID == discordID {
			userTagNumber = &entry.TagNumber
		}
		if entry.TagNumber == tagNumber {
			isTagAvailable = false
		}
	}

	return isTagAvailable, userTagNumber, nil

}

// InitiateManualTagSwap initiates a manual tag swap.
func (s *LeaderboardService) InitiateManualTagSwap(ctx context.Context, discordID string, tagNumber int) (string, error) {
	// Use GetTagInfo to check tag availability and get the user's current tag
	isTagAvailable, currentTagNumber, err := s.GetTagInfo(ctx, discordID, tagNumber)
	if err != nil {
		return "", fmt.Errorf("failed to check tag availability: %w", err)
	}

	if !isTagAvailable {
		// 1. Create or join a swap group (using UUID for group ID)
		swapGroupID := uuid.New().String()
		s.swapRequestsMu.Lock()
		if _, ok := s.swapRequests[swapGroupID]; !ok {
			s.swapRequests[swapGroupID] = make(map[int]tagSwapRequest)
		}
		s.swapRequestsMu.Unlock()

		// 2. Create a result channel and add the request to the group
		resultChan := make(chan error)
		s.swapRequestsMu.Lock()
		// No need to fetch currentTagNumber here as it's already retrieved from GetTagInfo

		// Ensure currentTagNumber is not nil before using it
		if currentTagNumber == nil {
			return "", fmt.Errorf("user %s not found in leaderboard", discordID)
		}

		s.swapRequests[swapGroupID][*currentTagNumber] = tagSwapRequest{
			discordID:  discordID,
			tagNumber:  tagNumber,
			resultChan: resultChan,
		}
		s.swapRequestsMu.Unlock()

		// 3. Check for matching requests and trigger the swap if found
		s.swapRequestsMu.Lock()
		if req, ok := s.swapRequests[swapGroupID][tagNumber]; ok {
			// Found a match! Perform the swap
			go s.performTagSwap(ctx, req.discordID, req.tagNumber, discordID, *currentTagNumber, req.resultChan, swapGroupID)
			// Remove both requests from the group after initiating the swap
			delete(s.swapRequests[swapGroupID], tagNumber)
			delete(s.swapRequests[swapGroupID], *currentTagNumber)
		} else {
			// No match yet, start a timeout goroutine
			go s.handleTagSwapTimeout(ctx, swapGroupID, discordID, tagNumber, resultChan)
		}

		// If there are 2 or more requests in the group, attempt to process matches
		if len(s.swapRequests[swapGroupID]) >= 2 {
			go s.processSwapGroup(ctx, swapGroupID)
		}
		s.swapRequestsMu.Unlock()

		// 4. Wait for the result or timeout
		select {
		case err := <-resultChan:
			if err != nil {
				return "", err
			}
			return "swap successful", nil
		case <-time.After(3 * time.Minute):
			return "", fmt.Errorf("tag swap timeout for group %s", swapGroupID)
		}
	} else {
		// Tag is available, directly update the leaderboard
		leaderboard, err := s.GetLeaderboard(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get leaderboard: %w", err)
		}

		// Update the tagNumber in the map directly
		leaderboard.LeaderboardData[tagNumber] = discordID

		// Convert the map to a slice of LeaderboardEntry
		var leaderboardData []structs.LeaderboardEntry
		for tag, discordID := range leaderboard.LeaderboardData {
			leaderboardData = append(leaderboardData, structs.LeaderboardEntry{
				DiscordID: discordID,
				TagNumber: tag,
			})
		}

		if err := s.updateLeaderboard(ctx, leaderboardData, structs.ServiceUpdateTagSourceManual); err != nil {
			return "", fmt.Errorf("failed to update leaderboard: %w", err)
		}

		return "tag updated successfully", nil
	}
}

// performTagSwap performs the actual tag swap between two users.
func (s *LeaderboardService) performTagSwap(ctx context.Context, discordID1 string, tagNumber1 int, discordID2 string, tagNumber2 int, resultChan chan error, swapGroupID string) {
	leaderboard, err := s.GetLeaderboard(ctx)
	if err != nil {
		resultChan <- fmt.Errorf("failed to get leaderboard: %w", err)
		return
	}

	// Swap the tag numbers in the map directly
	leaderboard.LeaderboardData[tagNumber1], leaderboard.LeaderboardData[tagNumber2] = leaderboard.LeaderboardData[tagNumber2], leaderboard.LeaderboardData[tagNumber1]

	// Convert the map to a slice of LeaderboardEntry
	var leaderboardData []structs.LeaderboardEntry
	for tag, discordID := range leaderboard.LeaderboardData {
		leaderboardData = append(leaderboardData, structs.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tag,
		})
	}

	if err := s.updateLeaderboard(ctx, leaderboardData, structs.ServiceUpdateTagSourceManual); err != nil {
		resultChan <- fmt.Errorf("failed to update leaderboard: %w", err)
		return
	}

	fmt.Printf("Swapped tag %d from user %s to user %s\n", tagNumber1, discordID1, discordID2)
	resultChan <- nil

	// After successful swap, check if there are any remaining requests in the group
	s.swapRequestsMu.Lock()
	defer s.swapRequestsMu.Unlock()

	delete(s.swapRequests[swapGroupID], tagNumber1)
	delete(s.swapRequests[swapGroupID], tagNumber2)

	if len(s.swapRequests[swapGroupID]) == 0 {
		delete(s.swapRequests, swapGroupID)
	} else {
		// If there are still requests in the group, try to process them again
		go s.processSwapGroup(ctx, swapGroupID)
	}
}

// handleTagSwapTimeout handles the timeout for a tag swap group.
func (s *LeaderboardService) handleTagSwapTimeout(ctx context.Context, swapGroupID string, discordID string, tagNumber int, resultChan chan error) {
	defer close(resultChan)

	select {
	case <-time.After(3 * time.Minute):
		s.swapRequestsMu.Lock()
		defer s.swapRequestsMu.Unlock()

		if _, ok := s.swapRequests[swapGroupID][tagNumber]; ok {
			fmt.Printf("Tag swap timeout for group %s. Unmatched user: %s\n", swapGroupID, discordID)
			delete(s.swapRequests[swapGroupID], tagNumber)
			if len(s.swapRequests[swapGroupID]) == 0 {
				delete(s.swapRequests, swapGroupID)
			}
			resultChan <- fmt.Errorf("tag swap timeout for group %s", swapGroupID)
		}
	case <-ctx.Done():
		s.swapRequestsMu.Lock()
		defer s.swapRequestsMu.Unlock()

		if _, ok := s.swapRequests[swapGroupID][tagNumber]; ok {
			delete(s.swapRequests[swapGroupID], tagNumber)
			if len(s.swapRequests[swapGroupID]) == 0 {
				delete(s.swapRequests, swapGroupID)
			}
		}
	}
}

// processSwapGroup processes a swap group to find and execute matching tag swap requests.
func (s *LeaderboardService) processSwapGroup(ctx context.Context, swapGroupID string) {
	s.swapRequestsMu.Lock()
	defer s.swapRequestsMu.Unlock()

	// Iterate over the requests in the group
	for tagNumber1, req1 := range s.swapRequests[swapGroupID] {
		if req2, ok := s.swapRequests[swapGroupID][req1.tagNumber]; ok {
			// Found a match! Perform the swap
			go s.performTagSwap(ctx, req1.discordID, req1.tagNumber, req2.discordID, req2.tagNumber, req1.resultChan, swapGroupID)
			// Remove both requests from the group after initiating the swap
			delete(s.swapRequests[swapGroupID], tagNumber1)
			delete(s.swapRequests[swapGroupID], req1.tagNumber)
		}
	}
}

// UpdateTag updates a user's tag in the leaderboard.
func (s *LeaderboardService) UpdateTag(ctx context.Context, discordID string, tagNumber int, source structs.ServiceUpdateTagSource) error { // Use the service-layer type
	if source == structs.ServiceUpdateTagSourceManual {
		return nil // No need to update for manual swaps as they are handled separately
	}

	leaderboard, err := s.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Convert the map to a slice of LeaderboardEntry
	var leaderboardData []structs.LeaderboardEntry
	for tag, discordID := range leaderboard.LeaderboardData {
		leaderboardData = append(leaderboardData, structs.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tag,
		})
	}

	found := false
	for i, entry := range leaderboardData {
		if entry.DiscordID == discordID {
			leaderboardData[i].TagNumber = tagNumber
			found = true
			break
		}
	}

	if !found {
		leaderboardData = append(leaderboardData, structs.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tagNumber,
		})
	}

	return s.updateLeaderboard(ctx, leaderboardData, source)
}

// UpdateLeaderboard updates the leaderboard with the processed and sorted leaderboard data.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, processedScores []structs.LeaderboardEntry, source structs.ServiceUpdateTagSource) error {

	return s.updateLeaderboard(ctx, processedScores, source)
}

// ProcessRanking processes the ranking data received from the Score module.
func (s *LeaderboardService) ProcessRanking(ctx context.Context, processedScores []structs.LeaderboardEntry) error {
	// Fetch the leaderboard data using the interface
	leaderboard, err := s.db.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Convert LeaderboardData from map[int]string to []structs.LeaderboardEntry
	var apiLeaderboardData []structs.LeaderboardEntry
	for tag, discordID := range leaderboard.LeaderboardData {
		apiLeaderboardData = append(apiLeaderboardData, structs.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tag,
		})
	}

	// Update tag numbers based on processed scores
	for i, entry := range processedScores {
		if i < len(apiLeaderboardData) && apiLeaderboardData[i].DiscordID == entry.DiscordID {
			apiLeaderboardData[i].TagNumber = entry.TagNumber
		}
	}

	// Convert LeaderboardData back to map[int]string
	leaderboardData := make(map[int]string)
	for _, entry := range apiLeaderboardData {
		leaderboardData[entry.TagNumber] = entry.DiscordID
	}

	// Use the interface to deactivate the current leaderboard
	if err := s.db.DeactivateCurrentLeaderboard(ctx); err != nil {
		return err
	}

	// Use the interface to insert the new leaderboard
	if err := s.db.InsertLeaderboard(ctx, leaderboardData, true); err != nil { // Pass leaderboardData and active status
		return err
	}

	return nil
}

// updateLeaderboard is a helper function to update the leaderboard with new data and source.
func (s *LeaderboardService) updateLeaderboard(ctx context.Context, leaderboardData []structs.LeaderboardEntry, source structs.ServiceUpdateTagSource) error { // Use structs.ServiceUpdateTagSource
	// 1. Fetch the active leaderboard
	leaderboard, err := s.db.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// 2. Convert LeaderboardData to []structs.LeaderboardEntry
	var currentLeaderboardData []structs.LeaderboardEntry
	for tag, discordID := range leaderboard.LeaderboardData {
		currentLeaderboardData = append(currentLeaderboardData, structs.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tag,
		})
	}

	// 3. Update the leaderboard data based on the source
	switch source {
	case structs.ServiceUpdateTagSourceProcessScores:
		tagIndex := 0
		for i := len(currentLeaderboardData) - 1; i >= 0; i-- {
			if currentLeaderboardData[i].TagNumber != 0 {
				if tagIndex < len(leaderboardData) {
					currentLeaderboardData[i].TagNumber = leaderboardData[tagIndex].TagNumber
					tagIndex++
				}
			}
		}
		leaderboardData = currentLeaderboardData // Update leaderboardData with the modified slice
	case structs.ServiceUpdateTagSourceManual, structs.ServiceUpdateTagSourceCreateUser:
		// No changes needed for manual or create user sources
	default:
		return errors.New("invalid update source")
	}

	// 4. Convert LeaderboardData to map[int]string for the database
	leaderboardDataMap := make(map[int]string)
	for _, entry := range leaderboardData {
		leaderboardDataMap[entry.TagNumber] = entry.DiscordID
	}

	// 5. Use the interface to update the leaderboard with a transaction
	err = s.db.UpdateLeaderboardWithTransaction(ctx, leaderboardDataMap) // Pass the map
	if err != nil {
		return fmt.Errorf("failed to update leaderboard with transaction: %w", err)
	}

	return nil
}
