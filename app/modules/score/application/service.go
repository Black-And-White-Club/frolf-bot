package scoreservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreService handles score processing logic.
type ScoreService struct {
	ScoreDB  scoredb.ScoreDB
	EventBus shared.EventBus
	logger   *slog.Logger
}

// NewScoreService creates a new ScoreService.
func NewScoreService(eventBus shared.EventBus, db scoredb.ScoreDB, logger *slog.Logger) *ScoreService {
	return &ScoreService{
		ScoreDB:  db,
		EventBus: eventBus,
		logger:   logger,
	}
}

// ProcessRoundScores processes scores received from the round module.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, event scoreevents.ScoresReceivedEvent) error {
	// 1. Convert and sort scores
	scores, err := s.prepareScores(event.Scores)
	if err != nil {
		return fmt.Errorf("error preparing scores: %w", err)
	}

	// 2. Log scores to the database
	if err := s.ScoreDB.LogScores(ctx, event.RoundID, scores, "auto"); err != nil {
		return fmt.Errorf("failed to log scores: %w", err)
	}

	// 3. Publish leaderboard update
	return s.publishLeaderboardUpdate(ctx, event.RoundID, scores)
}

// CorrectScore handles score corrections (manual updates).
func (s *ScoreService) CorrectScore(ctx context.Context, event scoreevents.ScoreCorrectedEvent) error {
	// 1. Update the score in the database
	tagNum, err := strconv.Atoi(event.TagNumber)
	if err != nil {
		return fmt.Errorf("invalid tag number: %w", err)
	}

	score := scoredb.Score{
		DiscordID: event.DiscordID,
		RoundID:   event.RoundID,
		Score:     event.NewScore,
		TagNumber: tagNum,
	}

	if err := s.ScoreDB.UpdateOrAddScore(ctx, &score); err != nil {
		return fmt.Errorf("failed to update/add score: %w", err)
	}

	// 2. Fetch, sort, and log updated scores
	scores, err := s.ScoreDB.GetScoresForRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to retrieve scores for round: %w", err)
	}

	sortedScores := s.sortScores(scores)
	if err := s.ScoreDB.LogScores(ctx, event.RoundID, sortedScores, "manual"); err != nil {
		return fmt.Errorf("failed to log updated scores: %w", err)
	}

	// 3. Publish leaderboard update
	return s.publishLeaderboardUpdate(ctx, event.RoundID, sortedScores)
}

// prepareScores converts, sorts, and returns scores for further processing.
func (s *ScoreService) prepareScores(eventScores []scoreevents.Score) ([]scoredb.Score, error) {
	var scores []scoredb.Score
	for _, score := range eventScores {
		tagNum, err := strconv.Atoi(score.TagNumber)
		if err != nil {
			return nil, fmt.Errorf("invalid tag number: %w", err)
		}
		scores = append(scores, scoredb.Score{
			DiscordID: score.DiscordID,
			Score:     score.Score,
			TagNumber: tagNum,
		})
	}
	return s.sortScores(scores), nil
}

// sortScores sorts scores by score (ascending) and tag number (descending).
func (s *ScoreService) sortScores(scores []scoredb.Score) []scoredb.Score {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].TagNumber > scores[j].TagNumber
		}
		return scores[i].Score < scores[j].Score
	})
	return scores
}

// publishLeaderboardUpdate publishes a leaderboard update event.
func (s *ScoreService) publishLeaderboardUpdate(ctx context.Context, roundID string, scores []scoredb.Score) error {
	var eventScores []scoreevents.Score
	for _, score := range scores {
		eventScores = append(eventScores, scoreevents.Score{
			DiscordID: score.DiscordID,
			Score:     score.Score,
			TagNumber: strconv.Itoa(score.TagNumber),
		})
	}

	event := scoreevents.LeaderboardUpdateEvent{
		RoundID: roundID,
		Scores:  eventScores,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard update event: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), eventData)
	msg.Metadata.Set("subject", scoreevents.LeaderboardUpdateEventSubject)

	if err := s.EventBus.Publish(ctx, scoreevents.LeaderboardStreamName, msg); err != nil {
		return fmt.Errorf("failed to publish leaderboard update event: %w", err)
	}

	s.logger.Info("Leaderboard update published", slog.String("round_id", roundID))
	return nil
}
