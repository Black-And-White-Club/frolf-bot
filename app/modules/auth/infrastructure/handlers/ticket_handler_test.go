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

	tests := []struct {
		name         string
		cookieValue  string
		setupService func(*FakeService)
		setupBus     func(*FakeEventBus) *[]string // returns pointer to collected topics
		verify       func(t *testing.T, rr *httptest.ResponseRecorder, publishedTopics *[]string)
	}{
		{
			name:        "success without sync requests",
			cookieValue: "old-token",
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
			cookieValue: "old-token",
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
			cookieValue: "old-token",
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
			cookieValue: "token",
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
