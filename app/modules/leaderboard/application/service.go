package leaderboardservice

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// LeaderboardService handles leaderboard logic.
type LeaderboardService struct {
	LeaderboardDB leaderboarddb.LeaderboardDB
	EventBus      shared.EventBus
	logger        *slog.Logger
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(db leaderboarddb.LeaderboardDB, eventBus shared.EventBus, logger *slog.Logger) *LeaderboardService {
	return &LeaderboardService{
		LeaderboardDB: db,
		EventBus:      eventBus,
		logger:        logger,
	}
}

// UpdateLeaderboard processes a LeaderboardUpdateEvent.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, event leaderboardevents.LeaderboardUpdateEvent) error {
	s.logger.Info("Updating leaderboard", "event", event)

	sortedScores, err := s.sortScores(event.Scores)
	if err != nil {
		s.logger.Error("Failed to sort scores", "error", err)
		return fmt.Errorf("failed to sort scores: %w", err)
	}

	entries := make(map[int]string)
	for _, score := range sortedScores {
		tagNumber, err := strconv.Atoi(score.TagNumber)
		if err != nil {
			return fmt.Errorf("failed to convert tag number to int: %w", err)
		}
		entries[tagNumber] = score.DiscordID
	}

	if err := s.LeaderboardDB.UpdateLeaderboard(ctx, entries); err != nil {
		s.logger.Error("Failed to update leaderboard in DB", "error", err)
		return &leaderboardevents.ErrLeaderboardUpdateFailed{Reason: err.Error()}
	}

	s.logger.Info("Leaderboard updated successfully")
	return nil
}

// AssignTag assigns a tag to a user.
func (s *LeaderboardService) AssignTag(ctx context.Context, event leaderboardevents.TagAssignedEvent) error {
	s.logger.Info("Assigning tag to user", "event", event)

	if err := s.LeaderboardDB.AssignTag(ctx, event.DiscordID, event.TagNumber); err != nil {
		s.logger.Error("Failed to assign tag", "error", err)
		return fmt.Errorf("failed to assign tag: %w", err)
	}

	s.logger.Info("Tag assigned successfully")
	return nil
}

// SwapTags swaps tags between two users.
func (s *LeaderboardService) SwapTags(ctx context.Context, requestorID, targetID string) error {
	s.logger.Info("Swapping tags", "requestor", requestorID, "target", targetID)

	if err := s.LeaderboardDB.SwapTags(ctx, requestorID, targetID); err != nil {
		s.logger.Error("Failed to swap tags", "error", err)
		return fmt.Errorf("failed to swap tags: %w", err)
	}

	s.logger.Info("Tags swapped successfully")
	return nil
}

// GetLeaderboard returns the current leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) ([]leaderboardevents.LeaderboardEntry, error) {
	s.logger.Debug("Fetching leaderboard")

	leaderboard, err := s.LeaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch leaderboard from DB", "error", err)
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	leaderboardData := leaderboard.LeaderboardData

	eventLeaderboard := make([]leaderboardevents.LeaderboardEntry, 0, len(leaderboardData))
	for tagNumber, discordID := range leaderboardData {
		eventLeaderboard = append(eventLeaderboard, leaderboardevents.LeaderboardEntry{
			TagNumber: strconv.Itoa(tagNumber),
			DiscordID: discordID,
		})
	}

	s.logger.Debug("Leaderboard fetched successfully")
	return eventLeaderboard, nil
}

// GetTagByDiscordID returns the tag number associated with a Discord ID.
func (s *LeaderboardService) GetTagByDiscordID(ctx context.Context, discordID string) (int, error) {
	s.logger.Debug("Fetching tag by Discord ID", "discordID", discordID)
	tagNumber, err := s.LeaderboardDB.GetTagByDiscordID(ctx, discordID)
	if err != nil {
		s.logger.Error("Failed to fetch tag by Discord ID", "error", err)
		return 0, fmt.Errorf("failed to get tag by Discord ID: %w", err)
	}
	s.logger.Debug("Tag fetched successfully", "tag", tagNumber)
	return tagNumber, nil
}

// CheckTagAvailability checks if a tag number is available.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Debug("Checking tag availability", "tagNumber", tagNumber)

	isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, tagNumber)
	if err != nil {
		s.logger.Error("Failed to check tag availability", "error", err)
		return false, fmt.Errorf("failed to check tag availability: %w", err)
	}

	s.logger.Debug("Tag availability checked", "tagNumber", tagNumber, "isAvailable", isAvailable)
	return isAvailable, nil
}

// sortScores sorts the scores according to your specific criteria.
func (s *LeaderboardService) sortScores(scores []leaderboardevents.Score) ([]leaderboardevents.Score, error) {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			tagI, errI := strconv.Atoi(scores[i].TagNumber)
			tagJ, errJ := strconv.Atoi(scores[j].TagNumber)
			if errI != nil || errJ != nil {
				// Log the error and use a default value for comparison
				s.logger.Error("Error converting tag number to integer", "error", errI, "error", errJ)
				return scores[i].TagNumber > scores[j].TagNumber
			}
			return tagI > tagJ
		}
		return scores[i].Score < scores[j].Score
	})

	return scores, nil
}
