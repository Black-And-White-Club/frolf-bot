package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Black-And-White-Club/tcr-bot/api/services"
	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/go-chi/chi/v5"
)

// CreateUserRequest represents the request body for creating a user.
type CreateUserRequest struct {
	Name      string `json:"name"`
	DiscordID string `json:"discordID"`
	Role      string `json:"role"`
}

// UpdateUserRequest represents the request body for updating a user.
type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Assuming 'userService' is a global variable or accessible within this package
var UserService *services.UserService

// CreateUser creates a new user.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	log.Println("CreateUser handler - Entering handler") // Add this line
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("CreateUser handler - failed to decode request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("CreateUser handler - received user: %+v", user)

	tagNumber, _ := strconv.Atoi(r.URL.Query().Get("tagNumber"))

	log.Printf("CreateUser handler - received tag: %+v", tagNumber)

	if err := UserService.CreateUser(r.Context(), &user, tagNumber); err != nil {
		log.Printf("CreateUser handler - failed to create user: %v", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetUser retrieves a user.
func GetUser(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	user, err := UserService.GetUser(r.Context(), discordID)
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
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	discordID := chi.URLParam(r, "discordID")

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get the current user to check the role
	currentUser, err := UserService.GetUser(r.Context(), discordID)
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

	err = UserService.UpdateUser(r.Context(), discordID, &models.User{
		Name: req.Name,
		Role: models.UserRole(req.Role),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UserRoutes sets up the routes for user management.
func UserRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", CreateUser)
	r.Get("/{discordID}", GetUser)
	r.Put("/{discordID}", UpdateUser)
	return r
}
