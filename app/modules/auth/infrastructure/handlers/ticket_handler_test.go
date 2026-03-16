package authhandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAuthHandlers_HandleHTTPLogin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name          string
		url           string
		body          string
		secureCookies bool
		setupService  func(*FakeService)
		verify        func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			url:  "/api/auth/callback",
			body: `{"token":"otp-token"}`,
			setupService: func(s *FakeService) {
				s.LoginUserFunc = func(ctx context.Context, oneTimeToken string) (*authservice.LoginResponse, error) {
					return &authservice.LoginResponse{
						RefreshToken: "valid-refresh-token",
						UserUUID:     "test-uuid",
					}, nil
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				// Check cookie
				cookies := rr.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == RefreshTokenCookie {
						found = true
						if c.Value != "valid-refresh-token" {
							t.Errorf("expected cookie value valid-refresh-token, got %s", c.Value)
						}
					}
				}
				if !found {
					t.Error("cookie not found")
				}
				// Check body
				var body map[string]string
				json.NewDecoder(rr.Body).Decode(&body)
				if body["user_uuid"] != "test-uuid" {
					t.Errorf("expected user_uuid test-uuid, got %s", body["user_uuid"])
				}
			},
		},
		{
			name: "success with secure cookies",
			url:  "/api/auth/callback",
			body: `{"token":"otp-token"}`,
			setupService: func(s *FakeService) {
				s.LoginUserFunc = func(ctx context.Context, oneTimeToken string) (*authservice.LoginResponse, error) {
					return &authservice.LoginResponse{
						RefreshToken: "valid-refresh-token",
						UserUUID:     "test-uuid",
					}, nil
				}
			},
			secureCookies: true,
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				cookies := rr.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == RefreshTokenCookie {
						found = true
						if !c.Secure {
							t.Error("expected cookie to be secure")
						}
					}
				}
				if !found {
					t.Error("cookie not found")
				}
			},
		},
		{
			name: "missing token",
			url:  "/api/auth/callback",
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected status 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "service error",
			url:  "/api/auth/callback",
			body: `{"token":"token"}`,
			setupService: func(s *FakeService) {
				s.LoginUserFunc = func(ctx context.Context, token string) (*authservice.LoginResponse, error) {
					return nil, errors.New("auth error")
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("expected status 401, got %d", rr.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(fakeService)
			}
			h := NewAuthHandlers(fakeService, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, tt.secureCookies, "", "")
			req := httptest.NewRequest(http.MethodPost, tt.url, bytes.NewBufferString(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			h.HandleHTTPLogin(rr, req)
			tt.verify(t, rr)
		})
	}
}

func TestAuthHandlers_HandleHTTPTicket(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	validCookie := strings.Repeat("0123456789abcdef", 4) // 64-char hex

	tests := []struct {
		name         string
		cookieValue  string
		setupService func(*FakeService)
		setupBus     func(*FakeEventBus) *[]string // returns pointer to collected topics
		verify       func(t *testing.T, rr *httptest.ResponseRecorder, publishedTopics *[]string)
	}{
		{
			name:        "success without sync requests",
			cookieValue: validCookie,
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(ctx context.Context, rt string, clubID string) (*authservice.TicketResponse, error) {
					return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: "rotated-token"}, nil
				}
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder, publishedTopics *[]string) {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				cookies := rr.Result().Cookies()
				for _, c := range cookies {
					if c.Name == RefreshTokenCookie && c.Value != "rotated-token" {
						t.Errorf("expected rotated cookie, got %s", c.Value)
					}
				}
				if publishedTopics != nil && len(*publishedTopics) != 0 {
					t.Errorf("expected no sync events published, got %d", len(*publishedTopics))
				}
			},
		},
		{
			name:        "dispatches sync requests on ticket",
			cookieValue: validCookie,
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(ctx context.Context, rt string, clubID string) (*authservice.TicketResponse, error) {
					return &authservice.TicketResponse{
						NATSToken:    "nats-jwt",
						RefreshToken: "rotated-token",
						SyncRequests: []authservice.SyncRequest{
							{UserID: "user-111", GuildID: "guild-222"},
							{UserID: "user-333", GuildID: "guild-444"},
						},
					}, nil
				}
			},
			setupBus: func(bus *FakeEventBus) *[]string {
				topics := &[]string{}
				bus.PublishFunc = func(topic string, msgs ...*message.Message) error {
					*topics = append(*topics, topic)
					return nil
				}
				return topics
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder, publishedTopics *[]string) {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				if publishedTopics == nil || len(*publishedTopics) != 2 {
					t.Fatalf("expected 2 sync events published, got %v", publishedTopics)
				}
				for i, topic := range *publishedTopics {
					if topic != "user.profile.sync.request.v1" {
						t.Errorf("published[%d] topic = %q, want user.profile.sync.request.v1", i, topic)
					}
				}
			},
		},
		{
			name:        "publish failure is tolerated (best-effort)",
			cookieValue: validCookie,
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(ctx context.Context, rt string, clubID string) (*authservice.TicketResponse, error) {
					return &authservice.TicketResponse{
						NATSToken:    "nats-jwt",
						RefreshToken: "rotated-token",
						SyncRequests: []authservice.SyncRequest{
							{UserID: "user-111", GuildID: "guild-222"},
						},
					}, nil
				}
			},
			setupBus: func(bus *FakeEventBus) *[]string {
				bus.PublishFunc = func(topic string, msgs ...*message.Message) error {
					return errors.New("bus unavailable")
				}
				return nil
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder, _ *[]string) {
				// Must still return 200 even if publish failed
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200 despite publish error, got %d", rr.Code)
				}
			},
		},
		{
			name:        "missing cookie",
			cookieValue: "",
			verify: func(t *testing.T, rr *httptest.ResponseRecorder, _ *[]string) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("expected status 401, got %d", rr.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(fakeService)
			}
			fakeBus := &FakeEventBus{}
			var publishedTopics *[]string
			if tt.setupBus != nil {
				publishedTopics = tt.setupBus(fakeBus)
			}
			h := NewAuthHandlers(fakeService, fakeBus, &FakeHelpers{}, logger, tracer, false, "", "")
			req := httptest.NewRequest("GET", "/api/auth/ticket", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: tt.cookieValue})
			}
			rr := httptest.NewRecorder()
			h.HandleHTTPTicket(rr, req)
			tt.verify(t, rr, publishedTopics)
		})
	}
}

// TestAuthHandlers_HandleHTTPTicket_Bearer tests the bearer-token path introduced
// for the Discord Activity flow. The bearer path differs from the cookie path in two ways:
// (1) it does not rotate a cookie, and (2) it includes "refresh_token" in the JSON response.
func TestAuthHandlers_HandleHTTPTicket_Bearer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	validBearer := strings.Repeat("0123456789abcdef", 4) // 64-char hex

	tests := []struct {
		name         string
		setupReq     func(r *http.Request)
		setupService func(*FakeService)
		wantStatus   int
		verifyBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "bearer_returns_refresh_token_in_json_body",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validBearer)
			},
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(_ context.Context, _, _ string) (*authservice.TicketResponse, error) {
					return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: "rotated-rt"}, nil
				}
			},
			wantStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				t.Helper()
				if body["refresh_token"] == nil || body["refresh_token"] == "" {
					t.Error("expected refresh_token in JSON response for bearer caller")
				}
				if body["ticket"] == nil {
					t.Error("expected ticket in JSON response")
				}
			},
		},
		{
			name: "cookie_path_does_not_return_refresh_token_in_json",
			setupReq: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: validBearer})
			},
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(_ context.Context, _, _ string) (*authservice.TicketResponse, error) {
					return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: "rotated-rt"}, nil
				}
			},
			wantStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				t.Helper()
				if _, ok := body["refresh_token"]; ok {
					t.Error("expected no refresh_token in JSON response for cookie caller")
				}
			},
		},
		{
			name: "cookie_takes_precedence_when_both_present",
			setupReq: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: validBearer})
				r.Header.Set("Authorization", "Bearer "+strings.Repeat("fedcba9876543210", 4))
			},
			setupService: func(s *FakeService) {
				s.GetTicketFunc = func(_ context.Context, rt, _ string) (*authservice.TicketResponse, error) {
					if rt != validBearer {
						return nil, errors.New("expected cookie token as refresh token, got: " + rt)
					}
					return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: "rotated-rt"}, nil
				}
			},
			wantStatus: http.StatusOK,
			// Cookie path: no refresh_token in JSON
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				if _, ok := body["refresh_token"]; ok {
					t.Error("expected no refresh_token in JSON — cookie path was used")
				}
			},
		},
		{
			name: "bearer_with_empty_value_after_prefix_returns_401",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer ")
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(svc)
			}
			h := NewAuthHandlers(svc, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, false, "", "")
			req := httptest.NewRequest(http.MethodGet, "/api/auth/ticket", nil)
			tt.setupReq(req)
			rr := httptest.NewRecorder()
			h.HandleHTTPTicket(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.verifyBody != nil && rr.Code == http.StatusOK {
				var result map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.verifyBody(t, result)
			}
		})
	}
}

func TestAuthHandlers_HandleHTTPLogout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name         string
		cookieValue  string
		setupService func(*FakeService)
		verify       func(t *testing.T, rr *httptest.ResponseRecorder, serviceCalled bool)
	}{
		{
			name:        "success",
			cookieValue: strings.Repeat("0123456789abcdef", 4),
			setupService: func(s *FakeService) {
				s.LogoutUserFunc = func(ctx context.Context, rt string) error { return nil }
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder, serviceCalled bool) {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				if !serviceCalled {
					t.Error("expected LogoutUser to be called")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceCalled := false
			fakeService := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(fakeService)
				// Wrap LogoutUserFunc to track call
				original := fakeService.LogoutUserFunc
				fakeService.LogoutUserFunc = func(ctx context.Context, rt string) error {
					serviceCalled = true
					if original != nil {
						return original(ctx, rt)
					}
					return nil
				}
			}
			h := NewAuthHandlers(fakeService, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, false, "", "")
			req := httptest.NewRequest("POST", "/api/auth/logout", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: tt.cookieValue})
			}
			rr := httptest.NewRecorder()
			h.HandleHTTPLogout(rr, req)
			tt.verify(t, rr, serviceCalled)
		})
	}
}
