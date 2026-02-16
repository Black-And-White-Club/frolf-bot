package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHealthEndpoints(t *testing.T) {
	// Setup minimal app
	app := &App{}
	app.HTTPRouter = chi.NewRouter()

	// Register health endpoints directly
	// This avoids initializing heavy dependencies like DB/NATS which would fail without a running environment
	app.registerHealthEndpoints()

	// We didn't fully initialize dependencies (DB, EventBus, etc.), 
	// so readiness checks should fail, but liveness should pass.

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "Liveness check /livez",
			path:           "/livez",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Health check /health (aliased to liveness)",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Readiness check /readyz (should fail due to missing deps)",
			path:           "/readyz",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			app.HTTPRouter.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
