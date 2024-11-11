package graph

import (
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
)

// Response structure for JSON responses
type Response struct {
	Message string `json:"message"`
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, code int, payload Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	data, _ := json.Marshal(payload) // Marshal to JSON without a newline
	w.Write(data)                    // Write the JSON directly
}

// SetupRoutes sets up the HTTP routes
func SetupRoutes(firestoreClient *firestore.Client) *chi.Mux {
	r := chi.NewRouter()

	// GraphQL endpoint
	r.Post("/v1/tcr", func(w http.ResponseWriter, r *http.Request) {
		// Handle GraphQL requests here
	})

	return r
}
