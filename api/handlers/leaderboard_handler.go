package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Black-And-White-Club/tcr-bot/api/services"
	"github.com/Black-And-White-Club/tcr-bot/api/structs"
	"github.com/go-chi/chi/v5"
)

// LeaderboardService is the service responsible for managing leaderboards.
var LeaderboardService *services.LeaderboardService

// GetLeaderboard retrieves the active leaderboard.
func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard, err := LeaderboardService.GetLeaderboard(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(leaderboard); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// GetUserTag retrieves the tag information for a user.
func GetUserTag(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	_, tagNumber, err := LeaderboardService.GetTagInfo(r.Context(), discordID, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user tag: %v", err), http.StatusInternalServerError)
		return
	}

	if tagNumber == nil {
		http.Error(w, "Tag not found for the provided discordID", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tagNumber); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// UpdateTagDto represents the input data for updating a tag.
type UpdateTagDto struct {
	TagNumber int `json:"tagNumber"`
}

// UpdateTag updates a user's tag.
func UpdateTag(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	var input UpdateTagDto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	_, err := LeaderboardService.InitiateManualTagSwap(r.Context(), discordID, input.TagNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update tag: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateLeaderboardDto represents the input data for updating the leaderboard.
type UpdateLeaderboardDto struct {
	LeaderboardData []structs.LeaderboardEntry `json:"leaderboardData"` // Use structs.LeaderboardEntry
	Source          structs.UpdateTagSource    `json:"source"`          // Use UpdateTagSource from structs
}

// UpdateLeaderboard updates the leaderboard with processed and sorted entries.
func UpdateLeaderboard(w http.ResponseWriter, r *http.Request) {
	var input UpdateLeaderboardDto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := LeaderboardService.UpdateLeaderboard(r.Context(), input.LeaderboardData, input.Source); err != nil { // Pass source to service
		http.Error(w, fmt.Sprintf("Failed to update leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// LeaderboardRoutes sets up the routes for the leaderboard controller.
func LeaderboardRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", GetLeaderboard)
	r.Get("/users/{discordID}/tag", GetUserTag)
	r.Put("/users/{discordID}/tag", UpdateTag)
	r.Post("/update", UpdateLeaderboard)
	return r
}
