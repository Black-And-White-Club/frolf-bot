// user/handlers.go
package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// UserHandlers defines the interface for user handlers.
type UserHandlers struct {
	commandService CommandService
	queryService   QueryService
}

// NewUserHandlers creates a new UserHandlers instance.
func NewUserHandlers(commandService CommandService, queryService QueryService) *UserHandlers {
	return &UserHandlers{
		commandService: commandService,
		queryService:   queryService,
	}
}

// CreateUser creates a new user.
func (h *UserHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest // Use the moved type
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tagNumber, _ := strconv.Atoi(r.URL.Query().Get("tagNumber"))

	user := db.User{
		Name:      req.Name,
		DiscordID: req.DiscordID,
		Role:      models.UserRole(req.Role),
	}

	if err := h.commandService.CreateUser(r.Context(), &user, tagNumber); err != nil { // Use commandService
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetUser retrieves a user.
func (h *UserHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	user, err := h.queryService.GetUserByID(r.Context(), discordID) // Use queryService
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// UpdateUser updates an existing user.
func (h *UserHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	var req models.UpdateUserRequest // Use the moved type
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get the current user to check the role
	currentUser, err := h.queryService.GetUserByID(r.Context(), discordID) // Use queryService
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}
	if currentUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Assuming you have access to the authenticated user's ID
	userID := r.Context().Value("userID").(string)

	// Prevent non-admin users from updating the role to Editor or Admin
	if (models.UserRole(req.Role) == models.UserRoleEditor || models.UserRole(req.Role) == models.UserRoleAdmin) && currentUser.Role != models.UserRoleAdmin && userID != discordID {
		http.Error(w, "Unauthorized to update role to Editor or Admin", http.StatusForbidden)
		return
	}

	err = h.commandService.UpdateUser(r.Context(), discordID, &db.User{
		Name: req.Name,
		Role: models.UserRole(req.Role),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
