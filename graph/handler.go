// handler.go
package graph

import (
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"github.com/romero-jace/tcr-bot/graph/services" // Adjust the import path
)

// Response represents a standard JSON response structure
type Response struct {
	Message string `json:"message"`
}

// ExampleHandler is a sample HTTP handler that requires authentication
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	user := services.ForContext(r.Context())
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var message string
	if user.IsAdmin() {
		message = "Hello Admin! You have full access."
	} else if user.IsEditor() {
		message = "Hello Editor! You can edit the leaderboard."
	} else {
		message = "Hello User! You can view the leaderboard."
	}

	respondWithJSON(w, http.StatusOK, Response{Message: message})
}

// respondWithError sends a JSON error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	respondWithJSON(w, code, Response{Message: message})
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// SetupRoutes sets up the HTTP routes and applies middleware
func SetupRoutes(firestoreClient *firestore.Client) *chi.Mux {
	router := chi.NewRouter()

	// Apply the middleware to the ExampleHandler
	router.With(services.Middleware(firestoreClient)).Get("/example", ExampleHandler)

	// Add more routes as needed
	// Example: router.With(services.Middleware(firestoreClient)).Post("/another", AnotherHandler)

	return router
}
