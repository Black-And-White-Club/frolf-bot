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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleActivityTokenExchange(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name         string
		method       string
		body         string
		setupService func(*FakeService)
		wantStatus   int
		verifyBody   func(t *testing.T, body map[string]string)
	}{

		{
			name:       "missing_code_empty_json",
			method:     http.MethodPost,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid_json",
			method:     http.MethodPost,
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "oauth_callback_error_returns_401",
			method: http.MethodPost,
			body:   `{"code":"bad-code"}`,
			setupService: func(s *FakeService) {
				s.HandleOAuthCallbackFunc = func(_ context.Context, _, _, _ string) (*authservice.LoginResponse, error) {
					return nil, errors.New("discord rejected code")
				}
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "get_ticket_error_returns_500",
			method: http.MethodPost,
			body:   `{"code":"valid-code"}`,
			setupService: func(s *FakeService) {
				s.HandleOAuthCallbackFunc = func(_ context.Context, _, _, _ string) (*authservice.LoginResponse, error) {
					return &authservice.LoginResponse{RefreshToken: "rt", UserUUID: "uuid-1"}, nil
				}
				s.GetTicketFunc = func(_ context.Context, _, _ string) (*authservice.TicketResponse, error) {
					return nil, errors.New("ticket service unavailable")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "success_returns_refresh_token_ticket_and_user_uuid",
			method: http.MethodPost,
			body:   `{"code":"valid-code"}`,
			setupService: func(s *FakeService) {
				s.HandleOAuthCallbackFunc = func(_ context.Context, provider, code, _ string) (*authservice.LoginResponse, error) {
					if provider != "discord-activity" {
						return nil, errors.New("unexpected provider: " + provider)
					}
					return &authservice.LoginResponse{RefreshToken: "my-refresh-token", UserUUID: "user-uuid-abc"}, nil
				}
				s.GetTicketFunc = func(_ context.Context, rt, _ string) (*authservice.TicketResponse, error) {
					if rt != "my-refresh-token" {
						return nil, errors.New("unexpected refresh token")
					}
					return &authservice.TicketResponse{NATSToken: "nats-jwt-xyz", RefreshToken: "rotated-refresh"}, nil
				}
			},
			wantStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]string) {
				t.Helper()
				if body["refresh_token"] != "rotated-refresh" {
					t.Errorf("refresh_token = %q, want %q", body["refresh_token"], "rotated-refresh")
				}
				if body["ticket"] != "nats-jwt-xyz" {
					t.Errorf("ticket = %q, want %q", body["ticket"], "nats-jwt-xyz")
				}
				if body["user_uuid"] != "user-uuid-abc" {
					t.Errorf("user_uuid = %q, want %q", body["user_uuid"], "user-uuid-abc")
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
			h := NewAuthHandlers(svc, &FakeEventBus{}, &FakeHelpers{}, logger, tracer, false, "", "")

			req := httptest.NewRequest(tt.method, "/api/auth/discord-activity/token-exchange", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.HandleActivityTokenExchange(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.verifyBody != nil && rr.Code == http.StatusOK {
				var result map[string]string
				if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				tt.verifyBody(t, result)
			}
		})
	}
}
