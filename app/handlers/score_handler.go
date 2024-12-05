package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/app/services"
	"github.com/go-chi/chi/v5"
)

// ScoreService is the service responsible for managing scores.
var ScoreService *services.ScoreService

// GetScore retrieves a specific score for a user and round.
func GetScore(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")
	discordID := chi.URLParam(r, "discordID")

	score, err := ScoreService.GetUserScore(r.Context(), discordID, roundID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user score: %v", err), http.StatusInternalServerError)
		return
	}

	if score == nil {
		http.Error(w, "Score not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(score); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// GetScoresForRound retrieves all scores for a specific round.
func GetScoresForRound(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")

	scores, err := ScoreService.GetScoresForRound(r.Context(), roundID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get scores for round: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// UpdateScoreDto represents the data transfer object for updating a score.
type UpdateScoreDto struct {
	Score     int  `json:"score"`
	TagNumber *int `json:"tagNumber"`
}

// UpdateScore updates a specific score for a user and round.
func UpdateScore(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")
	discordID := chi.URLParam(r, "discordID")

	var input UpdateScoreDto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	score, err := ScoreService.UpdateScore(r.Context(), roundID, discordID, input.Score, input.TagNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update score: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(score); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// ProcessScoresDto represents the data transfer object for processing multiple scores.
type ProcessScoresDto struct {
	RoundID int64               `json:"roundID"`
	Scores  []models.ScoreInput `json:"scores"` // Use models.ScoreInput
}

// ProcessScores processes a batch of scores.
func ProcessScores(w http.ResponseWriter, r *http.Request) {
	var input ProcessScoresDto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := ScoreService.ProcessScores(r.Context(), input.RoundID, input.Scores); err != nil {
		http.Error(w, fmt.Sprintf("Failed to process scores: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ScoreRoutes sets up the routes for the score controller.
func ScoreRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{roundID}/{discordID}", GetScore)
	r.Get("/{roundID}", GetScoresForRound)
	r.Put("/{roundID}/{discordID}", UpdateScore)
	r.Post("/process", ProcessScores)
	return r
}
