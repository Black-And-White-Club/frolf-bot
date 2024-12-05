package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/app/services"
	"github.com/go-chi/chi/v5"
)

// RoundService is the service responsible for managing rounds.
var RoundService *services.RoundService

// GetRounds retrieves all rounds.
func GetRounds(w http.ResponseWriter, r *http.Request) {
	rounds, err := RoundService.GetRounds(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch rounds: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rounds); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// GetRound retrieves a specific round by ID.
func GetRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	round, err := RoundService.GetRound(r.Context(), roundID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch round: %v", err), http.StatusInternalServerError)
		return
	}
	if round == nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(round); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// ScheduleRound schedules a new round.
func ScheduleRound(w http.ResponseWriter, r *http.Request) {
	var input models.ScheduleRoundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	round, err := RoundService.ScheduleRound(r.Context(), input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to schedule round: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(round); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// JoinRound adds a participant to a round.
func JoinRound(w http.ResponseWriter, r *http.Request) {
	var input models.JoinRoundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	round, err := RoundService.JoinRound(r.Context(), input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to join round: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(round); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// EditRound updates an existing round.
func EditRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	var input models.EditRoundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(string) // Get the user ID from the context

	round, err := RoundService.EditRound(r.Context(), roundID, userID, input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to edit round: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(round); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// DeleteRound deletes a round by ID.
func DeleteRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(string)

	if err := RoundService.DeleteRound(r.Context(), roundID, userID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete round: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SubmitScore submits a score for a participant in a round.
func SubmitScore(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	var input models.SubmitScoreInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	_, err = RoundService.SubmitScore(r.Context(), models.SubmitScoreInput{ // Use models.SubmitScoreInput
		RoundID:   roundID, // Use roundID here
		DiscordID: input.DiscordID,
		Score:     input.Score,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit score: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// FinalizeRound finalizes a round.
func FinalizeRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	_, err = RoundService.FinalizeAndProcessScores(r.Context(), roundID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to finalize round: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RoundRoutes sets up the routes for the round controller.
func RoundRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", GetRounds)
	r.Get("/{roundID}", GetRound)
	r.Post("/", ScheduleRound)
	r.Post("/join", JoinRound)
	r.Put("/{roundID}", EditRound)
	r.Post("/{roundID}/scores", SubmitScore)
	r.Post("/{roundID}/finalize", FinalizeRound)
	r.Delete("/{roundID}", DeleteRound) // Add the DeleteRound handler
	return r
}
