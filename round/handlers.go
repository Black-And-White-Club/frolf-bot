package round

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// RoundHandlers handles HTTP requests for rounds.
type RoundHandlers struct {
	commandService *RoundCommandService
	queryService   *RoundQueryService
}

// NewRoundHandlers creates a new RoundHandlers instance.
func NewRoundHandlers(commandService *RoundCommandService, queryService *RoundQueryService) *RoundHandlers {
	return &RoundHandlers{
		commandService: commandService,
		queryService:   queryService,
	}
}

// GetRounds retrieves all rounds.
func (h *RoundHandlers) GetRounds(w http.ResponseWriter, r *http.Request) {
	rounds, err := h.queryService.GetRounds(r.Context())
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
func (h *RoundHandlers) GetRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	round, err := h.queryService.GetRound(r.Context(), roundID)
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
func (h *RoundHandlers) ScheduleRound(w http.ResponseWriter, r *http.Request) {
	var input ScheduleRoundInput // Use the API model from round/models.go
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	round, err := h.commandService.ScheduleRound(r.Context(), input)
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
func (h *RoundHandlers) JoinRound(w http.ResponseWriter, r *http.Request) {
	var input JoinRoundInput // Use the API model from round/models.go
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	round, err := h.commandService.JoinRound(r.Context(), input)
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
func (h *RoundHandlers) EditRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	var input EditRoundInput // Use the API model from round/models.go
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(string) // Get the user ID from the context

	round, err := h.commandService.EditRound(r.Context(), roundID, userID, input)
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
func (h *RoundHandlers) DeleteRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	if err := h.commandService.DeleteRound(r.Context(), roundID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete round: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SubmitScore submits a score for a participant in a round.
func (h *RoundHandlers) SubmitScore(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	var input SubmitScoreInput // Use the API model from round/models.go
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	err = h.commandService.SubmitScore(r.Context(), SubmitScoreInput{
		RoundID:   roundID,
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
func (h *RoundHandlers) FinalizeRound(w http.ResponseWriter, r *http.Request) {
	roundIDStr := chi.URLParam(r, "roundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	_, err = h.commandService.FinalizeAndProcessScores(r.Context(), roundID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to finalize round: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RoundRoutes sets up the routes for the round controller.
func RoundRoutes(handlers *RoundHandlers) chi.Router {
	r := chi.NewRouter()
	r.Get("/", handlers.GetRounds)
	r.Get("/{roundID}", handlers.GetRound)
	r.Post("/", handlers.ScheduleRound)
	r.Post("/join", handlers.JoinRound)
	r.Put("/{roundID}", handlers.EditRound)
	r.Post("/{roundID}/scores", handlers.SubmitScore)
	r.Post("/{roundID}/finalize", handlers.FinalizeRound)
	r.Delete("/{roundID}", handlers.DeleteRound)
	return r
}
