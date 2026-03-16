package authhandlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// validToken returns a syntactically valid 64-char lowercase hex refresh token.
func validToken() string {
	return strings.Repeat("0123456789abcdef", 4) // exactly 64 hex chars
}

func TestAuthMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := AuthMiddleware(next)

	tests := []struct {
		name       string
		setupReq   func(r *http.Request)
		wantStatus int
	}{
		{
			name: "valid_cookie_passes",
			setupReq: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "refresh_token", Value: validToken()})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid_bearer_passes",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken())
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "neither_cookie_nor_bearer_rejected",
			setupReq:   func(r *http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid_cookie_format_falls_through_to_valid_bearer",
			setupReq: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "short"})
				r.Header.Set("Authorization", "Bearer "+validToken())
			},
			// Cookie format not validated in extractRefreshToken — but AuthMiddleware
			// checks isValidTokenFormat on the cookie value before passing through.
			// Short cookie fails validation, bearer is valid → passes.
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid_bearer_format_rejected",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer short-token")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "both_present_valid_cookie_takes_precedence",
			setupReq: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "refresh_token", Value: validToken()})
				r.Header.Set("Authorization", "Bearer "+validToken())
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			tt.setupReq(req)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://discord.com"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORSMiddleware(allowed)(next)

	t.Run("preflight_OPTIONS_allowed_origin_returns_200_with_PATCH_DELETE_methods", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/betting/bets", nil)
		req.Header.Set("Origin", "https://app.example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("OPTIONS status = %d, want 200", rr.Code)
		}
		methods := rr.Header().Get("Access-Control-Allow-Methods")
		if !strings.Contains(methods, "PATCH") {
			t.Errorf("Allow-Methods %q missing PATCH", methods)
		}
		if !strings.Contains(methods, "DELETE") {
			t.Errorf("Allow-Methods %q missing DELETE", methods)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
			t.Errorf("Access-Control-Allow-Origin = %q", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("unknown_origin_gets_no_CORS_headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/betting/overview", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("expected no CORS headers for unknown origin, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("no_origin_header_passthrough", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/betting/overview", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})
}
