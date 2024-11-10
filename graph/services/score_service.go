// graph/services/score_service.go
package services

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
)

type ScoreService struct {
	client *firestore.Client
	DB     *firestore.Client
}

// NewScoreService creates a new instance of ScoreService
func NewScoreService(client *firestore.Client) *ScoreService {
	return &ScoreService{
		client: client,
	}
}

// SubmitScore allows users to submit scores for themselves or multiple people
func (s *ScoreService) SubmitScore(ctx context.Context, round *model.Round, scores map[string]string) error {
	// Update scores for accepted participants
	for userID, scoreStr := range scores {
		// Trim spaces and validate the score format
		scoreStr = strings.TrimSpace(scoreStr)

		// Validate score format
		if !isValidGolfScore(scoreStr) {
			return errors.New("invalid score format for user " + userID)
		}

		// Convert score to an integer
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			return err // This should not happen due to the validation above
		}

		for _, participant := range round.Participants {
			if participant.User.ID == userID && (participant.Response == model.ACCEPT || participant.Response == model.TENTATIVE) {
				// Find or create a score entry for the user
				found := false
				for _, existingScore := range round.Scores {
					if existingScore.UserID == userID {
						existingScore.Score = score // Update existing score
						found = true
						break
					}
				}
				if !found {
					// If no existing score found, add a new score entry
					round.Scores = append(round.Scores, &model.Score{UserID: userID, Score: score})
				}
			}
		}
	}
	return nil
}

// ProcessScoring calculates rankings based on scores
func (s *ScoreService) ProcessScoring(round *model.Round) error {
	// Prepare a list for ranking
	type ParticipantScore struct {
		UserID string
		Score  int
	}

	var scores []ParticipantScore

	// Collect scores for participants who submitted scores
	for _, participant := range round.Participants {
		score := 0 // Default score if not submitted
		for _, submittedScore := range round.Scores {
			if submittedScore.UserID == participant.User.ID {
				score = submittedScore.Score
				break
			}
		}
		scores = append(scores, ParticipantScore{
			UserID: participant.User.ID,
			Score:  score,
		})
	}

	// Sort scores based on score (lower is better for golf)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	// Update rankings based on scores
	for rank, participantScore := range scores {
		// Update the participant's ranking in the round
		for _, participant := range round.Participants {
			if participant.User.ID == participantScore.UserID {
				participant.Rank = rank + 1 // Rank is 1-based
				break
			}
		}
	}

	return nil
}

// isValidGolfScore checks if the score is a valid golf score format
func isValidGolfScore(score string) bool {
	// Valid formats: -5, +10, 0, 5, 10 (with optional leading + or -)
	if len(score) == 0 {
		return false
	}
	if score[0] == '+' || score[0] == '-' {
		score = score[1:] // Remove the sign for validation
	}
	_, err := strconv.Atoi(score)
	return err == nil
}

// GetUser Score retrieves the score for a specific user from Firestore
func (s *ScoreService) GetUserScore(ctx context.Context, userID string) (int, error) {
	doc, err := s.DB.Collection("scores").Doc(userID).Get(ctx)
	if err != nil {
		return 0, err
	}

	var data struct {
		Score int `firestore:"score"`
	}
	if err := doc.DataTo(&data); err != nil {
		return 0, err
	}

	return data.Score, nil
}

// EditScore allows a user to edit their previously submitted score
func (s *ScoreService) EditScore(ctx context.Context, round *model.Round, userID string, newScoreStr string) error {
	// Trim spaces and validate the new score format
	newScoreStr = strings.TrimSpace(newScoreStr)

	// Validate new score format
	if !isValidGolfScore(newScoreStr) {
		return errors.New("invalid score format for user " + userID)
	}

	// Convert new score to an integer
	newScore, err := strconv.Atoi(newScoreStr)
	if err != nil {
		return err // This should not happen due to the validation above
	}

	// Find and update the score for the user
	for _, existingScore := range round.Scores {
		if existingScore.UserID == userID {
			existingScore.Score = newScore // Update existing score
			return nil
		}
	}

	// If the user does not have a score, return an error
	return errors.New("no existing score found for user " + userID)
}
