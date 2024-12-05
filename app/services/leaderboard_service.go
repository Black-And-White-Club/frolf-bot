package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
	"github.com/google/uuid"
)

// LeaderboardService handles leaderboard-related logic and database interactions.
type LeaderboardService struct {
	db                 db.LeaderboardDB
	natsConnectionPool *nats.NatsConnectionPool
	swapRequests       map[string]map[int]tagSwapRequest
	swapRequestsMu     sync.Mutex
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(db db.LeaderboardDB, natsConnectionPool *nats.NatsConnectionPool) *LeaderboardService {
	return &LeaderboardService{
		db:                 db,
		natsConnectionPool: natsConnectionPool,
		swapRequests:       make(map[string]map[int]tagSwapRequest),
	}
}

// GetLeaderboard retrieves the active leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) (*models.Leaderboard, error) {
	return s.db.GetLeaderboard(ctx)
}

// GetUserLeaderboardEntry retrieves the leaderboard entry for a user.
func (s *LeaderboardService) GetUserLeaderboardEntry(ctx context.Context, discordID string) (*models.LeaderboardEntry, error) {
	leaderboard, err := s.db.GetUserTag(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tag: %w", err)
	}
	if leaderboard == nil {
		return nil, nil // User not found in the leaderboard
	}

	for _, entry := range leaderboard.LeaderboardData {
		if entry.DiscordID == discordID {
			return &entry, nil
		}
	}

	return nil, nil // User not found in the leaderboard data
}

// CheckTagAvailability checks if a tag number is available.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	return s.db.IsTagAvailable(ctx, tagNumber)
}

// GetTagNumber retrieves the tag number for a user.
func (s *LeaderboardService) GetTagNumber(ctx context.Context, discordID string) (*int, error) {
	entry, err := s.GetUserLeaderboardEntry(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user leaderboard entry: %w", err)
	}
	if entry == nil {
		return nil, nil // User not found in the leaderboard
	}
	return &entry.TagNumber, nil
}

// tagSwapRequest represents a request to swap tags.
type tagSwapRequest struct {
	discordID  string
	tagNumber  int
	resultChan chan error
}

// InitiateManualTagSwap initiates a manual tag swap.
func (s *LeaderboardService) InitiateManualTagSwap(ctx context.Context, discordID string, tagNumber int) (string, error) {
	// Check if the tag is already available
	isTagAvailable, err := s.CheckTagAvailability(ctx, tagNumber)
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
		currentTagNumber := 0
		leaderboard, err := s.GetLeaderboard(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get leaderboard: %w", err)
		}
		for _, entry := range leaderboard.LeaderboardData {
			if entry.DiscordID == discordID {
				currentTagNumber = entry.TagNumber
				break
			}
		}
		s.swapRequests[swapGroupID][currentTagNumber] = tagSwapRequest{
			discordID:  discordID,
			tagNumber:  tagNumber,
			resultChan: resultChan,
		}
		s.swapRequestsMu.Unlock()

		// 3. Check for matching requests and trigger the swap if found
		s.swapRequestsMu.Lock()
		if req, ok := s.swapRequests[swapGroupID][tagNumber]; ok {
			// Found a match! Perform the swap
			go s.performTagSwap(ctx, req.discordID, req.tagNumber, discordID, currentTagNumber, req.resultChan, swapGroupID)
			// Remove both requests from the group after initiating the swap
			delete(s.swapRequests[swapGroupID], tagNumber)
			delete(s.swapRequests[swapGroupID], currentTagNumber)
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

		found := false
		for i, entry := range leaderboard.LeaderboardData {
			if entry.DiscordID == discordID {
				leaderboard.LeaderboardData[i].TagNumber = tagNumber
				found = true
				break
			}
		}

		if !found {
			leaderboard.LeaderboardData = append(leaderboard.LeaderboardData, models.LeaderboardEntry{
				DiscordID: discordID,
				TagNumber: tagNumber,
			})
		}

		if err := s.updateLeaderboard(ctx, leaderboard.LeaderboardData, models.UpdateTagSourceManual); err != nil {
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

	for i, entry := range leaderboard.LeaderboardData {
		if entry.DiscordID == discordID1 {
			leaderboard.LeaderboardData[i].TagNumber = tagNumber2
		} else if entry.DiscordID == discordID2 {
			leaderboard.LeaderboardData[i].TagNumber = tagNumber1
		}
	}

	if err := s.updateLeaderboard(ctx, leaderboard.LeaderboardData, models.UpdateTagSourceManual); err != nil {
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
func (s *LeaderboardService) UpdateTag(ctx context.Context, discordID string, tagNumber int, source models.UpdateTagSource) error {
	if source == models.UpdateTagSourceManual {
		return nil // No need to update for manual swaps as they are handled separately
	}

	leaderboard, err := s.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	found := false
	for i, entry := range leaderboard.LeaderboardData {
		if entry.DiscordID == discordID {
			leaderboard.LeaderboardData[i].TagNumber = tagNumber
			found = true
			break
		}
	}

	if !found {
		leaderboard.LeaderboardData = append(leaderboard.LeaderboardData, models.LeaderboardEntry{
			DiscordID: discordID,
			TagNumber: tagNumber,
		})
	}

	return s.updateLeaderboard(ctx, leaderboard.LeaderboardData, source)
}

// UpdateLeaderboard updates the leaderboard with the processed and sorted leaderboard data.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, processedScores []models.LeaderboardEntry, source models.UpdateTagSource) error {
	return s.updateLeaderboard(ctx, processedScores, source)
}

// StartNATSSubscribers starts the NATS subscribers for the leaderboard service.
func (s *LeaderboardService) StartNATSSubscribers(ctx context.Context) error {
	conn, err := s.natsConnectionPool.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer s.natsConnectionPool.ReleaseConnection(conn)

	// Subscribe to "check-tag-availability" subject
	_, err = conn.Subscribe("check-tag-availability", func(msg *nats.Msg) {
		var event nats.CheckTagAvailabilityEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshaling CheckTagAvailabilityEvent: %v", err)
			return
		}

		isAvailable, err := s.CheckTagAvailability(ctx, event.TagNumber)
		if err != nil {
			log.Printf("Error checking tag availability: %v", err)
			return
		}

		// Publish the response back to the user service
		response := &nats.TagAvailabilityResponse{
			IsAvailable: isAvailable,
		}
		responseData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshaling TagAvailabilityResponse: %v", err)
			return
		}

		err = conn.Publish(event.ReplyTo, responseData)
		if err != nil {
			log.Printf("Error publishing TagAvailabilityResponse: %v", err)
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to check-tag-availability: %w", err)
	}

	// Subscribe to "scores.processed" subject
	_, err = conn.Subscribe("scores.processed", func(msg *nats.Msg) {
		var event nats.ScoresProcessedEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshaling ScoresProcessedEvent: %v", err)
			return
		}

		// Convert []models.ScoreInput to []models.LeaderboardEntry
		var leaderboardEntries []models.LeaderboardEntry
		for _, score := range event.ProcessedScores {
			leaderboardEntries = append(leaderboardEntries, models.LeaderboardEntry{
				DiscordID: score.DiscordID,
				TagNumber: *score.TagNumber, // Assuming TagNumber is not nil
			})
		}

		err = s.ProcessRanking(ctx, leaderboardEntries) // Pass the converted slice
		if err != nil {
			log.Printf("Error processing ranking: %v", err)
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to scores.processed: %w", err)
	}

	// Subscribe to "user.created" subject
	_, err = conn.Subscribe("user.created", func(msg *nats.Msg) {
		var event nats.UserCreatedEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshaling UserCreatedEvent: %v", err)
			return
		}

		// Add the new user to the leaderboard using UpdateTag
		err = s.UpdateTag(ctx, event.DiscordID, event.TagNumber, models.UpdateTagSourceCreateUser)
		if err != nil {
			log.Printf("Error adding user to leaderboard: %v", err)
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user.created: %w", err)
	}

	// Subscribe to "leaderboard.get_tag_number" subject
	_, err = conn.Subscribe("leaderboard.get_tag_number", func(msg *nats.Msg) {
		var event nats.LeaderboardGetTagNumberEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshaling LeaderboardGetTagNumberEvent: %v", err)
			return
		}

		tagInfo, err := s.GetUserLeaderboardEntry(ctx, event.DiscordID)
		if err != nil {
			log.Printf("Error getting tag info: %v", err)
			return
		}

		// Publish the response
		var tagNumber *int
		if tagInfo != nil {
			tagNumber = &tagInfo.TagNumber
		}

		response := &nats.LeaderboardGetTagNumberResponse{
			TagNumber: tagNumber,
		}
		responseData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshaling LeaderboardGetTagNumberResponse: %v", err)
			return
		}

		err = conn.Publish(event.ReplyTo, responseData)
		if err != nil {
			log.Printf("Error publishing LeaderboardGetTagNumberResponse: %v", err)
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to leaderboard.get_tag_number: %w", err)
	}

	return nil
}

// ProcessRanking processes the ranking data received from the Score module.
func (s *LeaderboardService) ProcessRanking(ctx context.Context, processedScores []models.LeaderboardEntry) error {
	leaderboard, err := s.db.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	for i, entry := range processedScores {
		if i < len(leaderboard.LeaderboardData) && leaderboard.LeaderboardData[i].DiscordID == entry.DiscordID {
			leaderboard.LeaderboardData[i].TagNumber = entry.TagNumber
		}
	}

	if err := s.db.DeactivateCurrentLeaderboard(ctx); err != nil {
		return err
	}

	newLeaderboard := &models.Leaderboard{
		LeaderboardData: leaderboard.LeaderboardData,
		Active:          true,
	}
	if err := s.db.InsertLeaderboard(ctx, newLeaderboard); err != nil {
		return err
	}

	return nil
}

// updateLeaderboard is a helper function to update the leaderboard with new data and source.
func (s *LeaderboardService) updateLeaderboard(ctx context.Context, leaderboardData []models.LeaderboardEntry, source models.UpdateTagSource) error {
	leaderboard, err := s.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	switch source {
	case models.UpdateTagSourceProcessScores:
		tagIndex := 0
		for i := len(leaderboard.LeaderboardData) - 1; i >= 0; i-- {
			if leaderboard.LeaderboardData[i].TagNumber != 0 {
				if tagIndex < len(leaderboardData) {
					leaderboard.LeaderboardData[i].TagNumber = leaderboardData[tagIndex].TagNumber
					tagIndex++
				}
			}
		}
	case models.UpdateTagSourceManual, models.UpdateTagSourceCreateUser:
		leaderboard.LeaderboardData = leaderboardData
	default:
		return errors.New("invalid update source")
	}

	if err := s.db.DeactivateCurrentLeaderboard(ctx); err != nil {
		return err
	}

	// 4. Insert the new leaderboard
	newLeaderboard := &models.Leaderboard{
		LeaderboardData: leaderboard.LeaderboardData,
		Active:          true,
	}
	if err := s.db.InsertLeaderboard(ctx, newLeaderboard); err != nil {
		return err
	}

	return nil
}
