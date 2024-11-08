// graph/services/score_service.go
package services

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"your-module/graph/model"
	"cloud.google.com/go/firestore"
)

type ScoreService struct {
	client *firestore.Client
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
					round.Scores = append(round.Scores, &model.Score{User ID: userID, Score: score})
				}
			}
		}
	}
	return nil
}

// FinalizeRound finalizes a round and updates rankings
func (s *ScoreService) FinalizeRound(round *model.Round, editorID string) error {
	// Prepare a list for ranking
	type ParticipantScore struct {
		UserID string
		Score   int
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
		for i, participant := range round.Participants {
			if participant.User.ID == participantScore.UserID {
				participant.Rank = rank + 1 // Rank is 1-based
				break
			}
		}
	}

	// Log the edit action
	editLog := model.EditLog{
		EditorID:  editorID,
		Timestamp: time.Now(),
		Changes:   "Finalized round and updated rankings",
	}

	// Add the edit log to the round's edit history
	round.EditHistory = append(round.EditHistory, editLog)

	// Lock the round to prevent further changes
	round.IsFinalized = true

	return nil
}

// isValidGolfScore checks if the score is a valid golf score format
func isValidGolfScore(score string) bool {
	// Valid formats: -5, +10, 0, 5, 10 (with optional leading + or -)
	if len(score) == 0 {
		return false
	}
	if score[0] == '+' || score[0] == '-' {
		score = score[1:] // Remove leading sign for further validation
	}
	// Check if the remaining part is an integer
	for _, char := range score {
		if char < '0 || char > '9' {
			return false
		}
	}
	return true
}
