package authhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace/noop"

	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
)

// withProvider injects a chi URL parameter into the request context.
func withProvider(req *http.Request, provider string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", provider)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// findCookie searches the response recorder's cookies by name.
func findCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestAuthHandlers_HandleHTTPOAuthLinkInitiate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	const pwaBase = "https://app.example.com"

	tests := []struct {
		name          string
		secureCookies bool
		setupCookie   func(req *http.Request)
		setupService  func(*FakeService)
		verify        func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "no refresh_token cookie redirects to signin",
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase+"/auth/signin" {
					t.Errorf("expected redirect to %s/auth/signin, got %s", pwaBase, loc)
				}
			},
		},
		{
			name: "empty refresh_token cookie redirects to signin",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: ""})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase+"/auth/signin" {
					t.Errorf("expected redirect to %s/auth/signin, got %s", pwaBase, loc)
				}
			},
		},
		{
			name: "unsupported provider returns 400",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: "valid-token"})
			},
			setupService: func(s *FakeService) {
				s.InitiateOAuthLoginFunc = func(_ context.Context, _ string) (string, string, error) {
					return "", "", errors.New("unsupported provider")
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "success sets oauth_state and link_mode cookies then redirects",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: "valid-token"})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				// Redirect goes to the provider URL returned by the service.
				if loc := rr.Header().Get("Location"); loc == "" {
					t.Error("expected non-empty Location header")
				}
				stateCookie := findCookie(rr, oauthStateCookie)
				if stateCookie == nil {
					t.Fatal("expected oauth_state cookie to be set")
				}
				if !stateCookie.HttpOnly {
					t.Error("expected oauth_state cookie to be HttpOnly")
				}
				if stateCookie.Secure {
					t.Error("expected oauth_state cookie not to be Secure in insecure mode")
				}
				if stateCookie.MaxAge != 300 {
					t.Errorf("expected oauth_state MaxAge=300, got %d", stateCookie.MaxAge)
				}

				linkCookie := findCookie(rr, linkModeCookie)
				if linkCookie == nil {
					t.Fatal("expected link_mode cookie to be set")
				}
				if !linkCookie.HttpOnly {
					t.Error("expected link_mode cookie to be HttpOnly")
				}
				if linkCookie.MaxAge != 300 {
					t.Errorf("expected link_mode MaxAge=300, got %d", linkCookie.MaxAge)
				}
			},
		},
		{
			name:          "success with secure cookies sets Secure flag",
			secureCookies: true,
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: "valid-token"})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				stateCookie := findCookie(rr, oauthStateCookie)
				if stateCookie == nil {
					t.Fatal("expected oauth_state cookie")
				}
				if !stateCookie.Secure {
					t.Error("expected oauth_state cookie to have Secure=true")
				}

				linkCookie := findCookie(rr, linkModeCookie)
				if linkCookie == nil {
					t.Fatal("expected link_mode cookie")
				}
				if !linkCookie.Secure {
					t.Error("expected link_mode cookie to have Secure=true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(svc)
			}
			h := NewAuthHandlers(svc, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, tt.secureCookies, pwaBase, "")
			req := httptest.NewRequest(http.MethodGet, "/api/auth/discord/link", nil)
			req = withProvider(req, "discord")
			if tt.setupCookie != nil {
				tt.setupCookie(req)
			}
			rr := httptest.NewRecorder()
			h.HandleHTTPOAuthLinkInitiate(rr, req)
			tt.verify(t, rr)
		})
	}
}

func TestAuthHandlers_HandleHTTPOAuthCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	const pwaBase = "https://app.example.com"
	const testState = "test-csrf-state"
	const testCode = "test-auth-code"

	tests := []struct {
		name         string
		rawURL       string
		setupCookies func(req *http.Request)
		setupService func(*FakeService)
		verify       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:   "missing state cookie returns 400",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rr.Code)
				}
			},
		},
		{
			name:   "empty state query param returns 400",
			rawURL: "/api/auth/discord/callback",
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rr.Code)
				}
			},
		},
		{
			name:   "state mismatch returns 400",
			rawURL: "/api/auth/discord/callback?state=wrong-state&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rr.Code)
				}
			},
		},
		{
			name:   "missing code returns 400",
			rawURL: "/api/auth/discord/callback?state=" + testState,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rr.Code)
				}
			},
		},
		{
			name:   "link mode — no refresh_token redirects to account error",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
				req.AddCookie(&http.Cookie{Name: linkModeCookie, Value: "1"})
				// No refresh_token cookie
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase+"/account?error=link_failed" {
					t.Errorf("expected redirect to %s/account?error=link_failed, got %s", pwaBase, loc)
				}
			},
		},
		{
			name:   "link mode — LinkIdentityToUser error redirects to account error",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
				req.AddCookie(&http.Cookie{Name: linkModeCookie, Value: "1"})
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: "user-refresh-token"})
			},
			setupService: func(s *FakeService) {
				s.LinkIdentityToUserFunc = func(_ context.Context, _, _, _, _ string) error {
					return errors.New("link failed")
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase+"/account?error=link_failed" {
					t.Errorf("expected redirect to %s/account?error=link_failed, got %s", pwaBase, loc)
				}
			},
		},
		{
			name:   "link mode — success clears link_mode cookie and redirects to account",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
				req.AddCookie(&http.Cookie{Name: linkModeCookie, Value: "1"})
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: "user-refresh-token"})
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase+"/account" {
					t.Errorf("expected redirect to %s/account, got %s", pwaBase, loc)
				}
				// link_mode cookie should be cleared (MaxAge=-1)
				cleared := findCookie(rr, linkModeCookie)
				if cleared == nil {
					t.Fatal("expected link_mode cookie in response to be cleared")
				}
				if cleared.MaxAge != -1 {
					t.Errorf("expected link_mode MaxAge=-1 (cleared), got %d", cleared.MaxAge)
				}
			},
		},
		{
			name:   "login mode — callback error returns 401",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
			},
			setupService: func(s *FakeService) {
				s.HandleOAuthCallbackFunc = func(_ context.Context, _, _, _ string) (*authservice.LoginResponse, error) {
					return nil, errors.New("auth failed")
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("expected 401, got %d", rr.Code)
				}
			},
		},
		{
			name:   "login mode — success sets refresh_token cookie and redirects to PWA",
			rawURL: "/api/auth/discord/callback?state=" + testState + "&code=" + testCode,
			setupCookies: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: testState})
			},
			setupService: func(s *FakeService) {
				s.HandleOAuthCallbackFunc = func(_ context.Context, _, _, _ string) (*authservice.LoginResponse, error) {
					return &authservice.LoginResponse{RefreshToken: "new-refresh-token", UserUUID: "user-uuid"}, nil
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusFound {
					t.Errorf("expected 302, got %d", rr.Code)
				}
				if loc := rr.Header().Get("Location"); loc != pwaBase {
					t.Errorf("expected redirect to %s, got %s", pwaBase, loc)
				}
				refreshCookie := findCookie(rr, RefreshTokenCookie)
				if refreshCookie == nil {
					t.Fatal("expected refresh_token cookie to be set")
				}
				if refreshCookie.Value != "new-refresh-token" {
					t.Errorf("expected refresh_token=new-refresh-token, got %s", refreshCookie.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(svc)
			}
			h := NewAuthHandlers(svc, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, false, pwaBase, "")
			req := httptest.NewRequest(http.MethodGet, tt.rawURL, nil)
			req = withProvider(req, "discord")
			if tt.setupCookies != nil {
				tt.setupCookies(req)
			}
			rr := httptest.NewRecorder()
			h.HandleHTTPOAuthCallback(rr, req)
			tt.verify(t, rr)
		})
	}
}
