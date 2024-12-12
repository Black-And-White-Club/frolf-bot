package roundhandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	roundinterface "github.com/Black-And-White-Club/tcr-bot/round/commandsinterface"
	converter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	roundhelper "github.com/Black-And-White-Club/tcr-bot/round/helpers"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	"github.com/go-chi/chi/v5"
)

// RoundHandlers handles HTTP requests for rounds.
type RoundHandlers struct {
	roundDB        rounddb.RoundDB
	converter      converter.RoundConverter // Use the RoundConverter interface
	commandService roundinterface.CommandService
	queryService   roundqueries.QueryService
	roundHelper    roundhelper.RoundHelper
}

// NewRoundHandlers creates a new RoundHandlers instance.
func NewRoundHandlers(roundDB rounddb.RoundDB, converter converter.RoundConverter, commandService roundinterface.CommandService, queryService roundqueries.QueryService) *RoundHandlers {
	return &RoundHandlers{
		roundDB:        roundDB,
		converter:      converter,
		commandService: commandService,
		queryService:   queryService,
		roundHelper:    &roundhelper.RoundHelperImpl{Converter: converter}, // Initialize with the concrete type
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

	round, err := h.roundHelper.GetRound(r.Context(), h.roundDB, h.converter, roundID)
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
	var input apimodels.ScheduleRoundInput
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
	var input apimodels.JoinRoundInput
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

	var input apimodels.EditRoundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	DiscordID := r.Context().Value("DiscordID").(string)

	round, err := h.commandService.EditRound(r.Context(), roundID, DiscordID, input)
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

	var input apimodels.SubmitScoreInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	err = h.commandService.SubmitScore(r.Context(), apimodels.SubmitScoreInput{
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
func RoundRoutes(handlers RoundHandler) chi.Router {
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
