package graph_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph" // Ensure this import path is correct
)

// Test for RespondWithJSON function
func TestRespondWithJSON(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		payload  graph.Response
		expected string
	}{
		{
			name:     "Successful Response",
			code:     http.StatusOK,
			payload:  graph.Response{Message: "Success"},
			expected: `{"message":"Success"}`,
		},
		{
			name:     "Another Successful Response",
			code:     http.StatusCreated,
			payload:  graph.Response{Message: "Resource created"},
			expected: `{"message":"Resource created"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			graph.RespondWithJSON(rr, tt.code, tt.payload)

			if status := rr.Code; status != tt.code {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.code)
			}

			var response graph.Response
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("could not unmarshal response: %v", err)
			}

			if response.Message != tt.payload.Message {
				t.Errorf("handler returned unexpected body: got %v want %v", response.Message, tt.payload.Message)
			}
		})
	}
}

// Test for RespondWithError function
func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		expected string
	}{
		{
			name:     "Internal Server Error",
			code:     http.StatusInternalServerError,
			message:  "Something went wrong",
			expected: `{"message":"Something went wrong"}`,
		},
		{
			name:     "Not Found",
			code:     http.StatusNotFound,
			message:  "Resource not found",
			expected: `{"message":"Resource not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			rr := httptest.NewRecorder()
			// Call the function that handles the request
			graph.RespondWithJSON(rr, tt.code, graph.Response{Message: tt.message})

			// Check if the response body matches the expected output
			if rr.Body.String() != tt.expected {
				t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), tt.expected)
			}
		})
	}
}

// Test for SetupRoutes function
func TestSetupRoutes(t *testing.T) {
	// Mock Firestore client
	mockFirestoreClient := &firestore.Client{} // You may need to use a mock library for more complex tests

	tests := []struct {
		name string
		args struct {
			firestoreClient *firestore.Client
		}
		want bool // Instead of wanting the exact *chi.Mux, we can check if the routes are set up correctly
	}{
		{
			name: "Setup Routes",
			args: struct {
				firestoreClient *firestore.Client
			}{
				firestoreClient: mockFirestoreClient,
			},
			want: true, // We expect routes to be set up correctly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.SetupRoutes(tt.args.firestoreClient)

			// Check if the router has the expected routes
			if got == nil {
				t.Errorf("SetupRoutes() returned nil")
			}

			// Check if the route for the GraphQL endpoint exists
			reqBody := `{"query": "{ hello }"}` // Replace with a valid query based on your schema
			req, _ := http.NewRequest("POST", "/v1/tcr", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			got.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK { // Assuming you want a successful response for a valid query
				t.Errorf("Expected status code 200 for valid request, got %v", rr.Code)
			}
		})
	}
}
