// user/handlers.go
package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	usercommands "github.com/Black-And-White-Club/tcr-bot/user/commands"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	userqueries "github.com/Black-And-White-Club/tcr-bot/user/queries"
	"github.com/go-chi/chi/v5"
)

// UserHandlers defines the interface for user handlers.
type UserHandlers struct {
	commandService usercommands.UserService
	queryService   userqueries.QueryService
}

// NewUserHandlers creates a new UserHandlers instance.
func NewUserHandlers(commandService usercommands.UserService, queryService userqueries.QueryService) *UserHandlers {
	return &UserHandlers{
		commandService: commandService,
		queryService:   queryService,
	}
}

// CreateUser creates a new user.
func (h *UserHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req userapimodels.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tagNumber, _ := strconv.Atoi(r.URL.Query().Get("tagNumber")) // We'll need to handle potential errors here later

	if err := h.commandService.CreateUser(r.Context(), req.DiscordID, req.Name, req.Role, tagNumber); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetUser retrieves a user.
func (h *UserHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	user, err := h.queryService.GetUserByDiscordID(r.Context(), discordID)
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

	var req userapimodels.UpdateUserCommand // Use a regular UpdateUserRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get the current user to check the role
	currentUser, err := h.queryService.GetUserByDiscordID(r.Context(), discordID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}
	if currentUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Prevent non-admin users from updating the role to Editor or Admin
	if (userapimodels.UserRole(req.Role) == userapimodels.UserRoleEditor || userapimodels.UserRole(req.Role) == userapimodels.UserRoleAdmin) &&
		userapimodels.UserRole(currentUser.Role) != userapimodels.UserRoleAdmin {
		http.Error(w, "Unauthorized to update role to Editor or Admin", http.StatusForbidden)
		return
	}

	// Use the UpdateUserCommand from your api_models package
	cmd := userapimodels.UpdateUserCommand{
		DiscordID: discordID,
		// Role:      req.Role,  // This line is unnecessary
		Updates: req.Updates, // Assuming this is in your UpdateUserRequest
	}

	if err := h.commandService.UpdateUser(r.Context(), cmd.DiscordID, cmd.Updates); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserRole retrieves the role of a user.
func (h *UserHandlers) GetUserRole(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	role, err := h.queryService.GetUserRole(r.Context(), discordID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user role: %v", err), http.StatusInternalServerError)
		return
	}

	// Assuming UserRole is a string, otherwise adjust accordingly
	response := struct {
		Role string `json:"role"`
	}{
		Role: role,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// Routes returns the routes for the user module.
func UserRoutes(h UserHandler) chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateUser)
	r.Get("/{discordID}", h.GetUser)
	r.Put("/{discordID}", h.UpdateUser)
	r.Get("/{discordID}/role", h.GetUserRole)

	return r
}
