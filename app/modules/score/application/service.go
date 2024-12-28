package scoreservice

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreService handles score processing logic.
type ScoreService struct {
	ScoreDB    scoredb.ScoreDB
	Publisher  message.Publisher
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
}

// NewScoreService creates a new ScoreService.
func NewScoreService(publisher message.Publisher, subscriber message.Subscriber, db scoredb.ScoreDB, logger watermill.LoggerAdapter) *ScoreService {
	return &ScoreService{
		ScoreDB:    db,
		Publisher:  publisher,
		Subscriber: subscriber,
		logger:     logger,
	}
}

// ProcessRoundScores processes scores received from the round module.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, event scoreevents.ScoresReceivedEvent) error {
	// 1. Convert Scores from scoreevents.Score to scoredb.Score
	dbScores, err := s.convertToDBScores(event.Scores)
	if err != nil {
		return fmt.Errorf("failed to convert scores to db scores: %w", err)
	}

	// 2. Sort the scores
	sortedScores, err := s.sortScores(dbScores)
	if err != nil {
		return fmt.Errorf("failed to sort scores: %w", err)
	}

	// 3. Log the scores to the database
	if err := s.ScoreDB.LogScores(ctx, event.RoundID, sortedScores, "auto"); err != nil {
		return fmt.Errorf("failed to log scores: %w", err)
	}

	// 4. Convert back to scoreevents.Score for publishing
	eventScores, err := s.convertToEventScores(sortedScores)
	if err != nil {
		return fmt.Errorf("failed to convert scores to event scores: %w", err)
	}

	// 5. Publish LeaderboardUpdateEvent
	if err := s.publishLeaderboardUpdate(event.RoundID, eventScores); err != nil {
		return fmt.Errorf("failed to publish LeaderboardUpdateEvent: %w", err)
	}

	return nil
}

// CorrectScore handles score corrections (manual updates).
func (s *ScoreService) CorrectScore(ctx context.Context, event scoreevents.ScoreCorrectedEvent) error {
	// 1. Prepare the score for the database
	tagNum, err := strconv.Atoi(event.TagNumber)
	if err != nil {
		return fmt.Errorf("failed to convert tag number to int: %w", err)
	}

	score := &scoredb.Score{
		DiscordID: event.DiscordID,
		RoundID:   event.RoundID,
		Score:     event.NewScore,
		TagNumber: tagNum,
	}

	// 2. Update/add the score in the database
	if err := s.ScoreDB.UpdateOrAddScore(ctx, score); err != nil {
		return fmt.Errorf("failed to update/add score: %w", err)
	}

	// 3. Fetch updated scores from the database for the round
	updatedScores, err := s.ScoreDB.GetScoresForRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get updated scores: %w", err)
	}

	// 4. Sort the updated scores
	sortedScores, err := s.sortScores(updatedScores)
	if err != nil {
		return fmt.Errorf("failed to sort scores: %w", err)
	}

	// 5. Log updated scores (with "manual" source)
	err = s.ScoreDB.LogScores(ctx, event.RoundID, sortedScores, "manual")
	if err != nil {
		return fmt.Errorf("failed to log updated scores: %w", err)
	}

	// 6. Convert back to scoreevents.Score for publishing
	eventScores, err := s.convertToEventScores(sortedScores)
	if err != nil {
		return fmt.Errorf("failed to convert scores to event scores: %w", err)
	}

	// 7. Publish LeaderboardUpdateEvent with the corrected scores
	if err := s.publishLeaderboardUpdate(event.RoundID, eventScores); err != nil {
		return fmt.Errorf("failed to publish LeaderboardUpdateEvent: %w", err)
	}

	return nil
}

// sortScores sorts the scores according to your specific criteria.
func (s *ScoreService) sortScores(scores []scoredb.Score) ([]scoredb.Score, error) {
	// Sort by score (ascending), then by tag number (descending)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].TagNumber > scores[j].TagNumber // Sort by tag number in descending order
		}
		return scores[i].Score < scores[j].Score // Sort by score in ascending order
	})

	return scores, nil
}

// publishLeaderboardUpdate publishes a LeaderboardUpdateEvent.
func (s *ScoreService) publishLeaderboardUpdate(roundID string, scores []scoreevents.Score) error {
	// 1. Prepare the event
	evt := scoreevents.LeaderboardUpdateEvent{
		RoundID: roundID,
		Scores:  scores,
	}

	// 2. Marshal the event to JSON
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal LeaderboardUpdateEvent: %w", err)
	}

	// 3. Create a new message
	msg := message.NewMessage(watermill.NewUUID(), evtData)

	// 4. Publish the message
	if err := s.Publisher.Publish(scoreevents.LeaderboardUpdateEventSubject, msg); err != nil {
		return fmt.Errorf("failed to publish LeaderboardUpdateEvent: %w", err)
	}

	return nil
}

// convertToDBScores converts a slice of scoreevents.Score to scoredb.Score
func (s *ScoreService) convertToDBScores(eventScores []scoreevents.Score) ([]scoredb.Score, error) {
	dbScores := make([]scoredb.Score, len(eventScores))
	for i, score := range eventScores {
		tagNumber, err := strconv.Atoi(score.TagNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to convert tag number to int: %w", err)
		}
		dbScores[i] = scoredb.Score{
			DiscordID: score.DiscordID,
			Score:     score.Score,
			TagNumber: tagNumber,
		}
	}
	return dbScores, nil
}

// convertToEventScores converts a slice of scoredb.Score to scoreevents.Score
func (s *ScoreService) convertToEventScores(dbScores []scoredb.Score) ([]scoreevents.Score, error) {
	eventScores := make([]scoreevents.Score, len(dbScores))
	for i, score := range dbScores {
		eventScores[i] = scoreevents.Score{
			DiscordID: score.DiscordID,
			Score:     score.Score,
			TagNumber: strconv.Itoa(score.TagNumber),
		}
	}
	return eventScores, nil
}
