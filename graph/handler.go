// handler.go
package graph

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/services" // Adjust the import path
)

// ExampleHandler is a sample HTTP handler that requires authentication
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	user := services.ForContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.IsAdmin() {
		// Handle admin-specific logic
		w.Write([]byte("Hello Admin! You have full access."))
	} else if user.IsEditor() {
		// Handle editor-specific logic
		w.Write([]byte("Hello Editor! You can edit the leaderboard."))
	} else {
		// Handle normal user logic
		w.Write([]byte("Hello User! You can view the leaderboard."))
	}
}

// SetupRoutes sets up the HTTP routes and applies middleware
func SetupRoutes(firestoreClient *firestore.Client) *http.ServeMux {
	mux := http.NewServeMux()

	// Apply the middleware to the ExampleHandler
	mux.Handle("/example", services.Middleware(firestoreClient)(http.HandlerFunc(ExampleHandler)))

	// Add more routes as needed

	return mux
}
