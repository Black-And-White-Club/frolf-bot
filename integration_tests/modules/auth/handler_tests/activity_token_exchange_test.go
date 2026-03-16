// Package authhandlertests contains HTTP-level integration tests for the auth
// handler endpoints. Unlike unit tests (which call handlers directly), these
// tests start a real HTTP server using chi, making actual network requests
// through the full HTTP middleware stack.
package authhandlertests

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

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	authhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/handlers"
)

// ─── Minimal fakes ────────────────────────────────────────────────────────────

type integrationFakeService struct {
	handleOAuthCallbackFn func(ctx context.Context, provider, code, state string) (*authservice.LoginResponse, error)
	getTicketFn           func(ctx context.Context, refreshToken, clubID string) (*authservice.TicketResponse, error)
}

func (s *integrationFakeService) HandleOAuthCallback(ctx context.Context, provider, code, state string) (*authservice.LoginResponse, error) {
	if s.handleOAuthCallbackFn != nil {
		return s.handleOAuthCallbackFn(ctx, provider, code, state)
	}
	return &authservice.LoginResponse{RefreshToken: "rt-default", UserUUID: "uuid-default"}, nil
}
func (s *integrationFakeService) GetTicket(ctx context.Context, rt, clubID string) (*authservice.TicketResponse, error) {
	if s.getTicketFn != nil {
		return s.getTicketFn(ctx, rt, clubID)
	}
	return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: rt}, nil
}
func (s *integrationFakeService) GenerateMagicLink(_ context.Context, _, _ string, _ authdomain.Role) (*authservice.MagicLinkResponse, error) {
	return nil, nil
}
func (s *integrationFakeService) ValidateToken(_ context.Context, _ string) (*authdomain.Claims, error) {
	return nil, nil
}
func (s *integrationFakeService) HandleNATSAuthRequest(_ context.Context, _ *authservice.NATSAuthRequest) (*authservice.NATSAuthResponse, error) {
	return nil, nil
}
func (s *integrationFakeService) LoginUser(_ context.Context, _ string) (*authservice.LoginResponse, error) {
	return nil, nil
}
func (s *integrationFakeService) LogoutUser(_ context.Context, _ string) error { return nil }
func (s *integrationFakeService) InitiateOAuthLogin(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (s *integrationFakeService) LinkIdentityToUser(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *integrationFakeService) UnlinkProvider(_ context.Context, _, _ string) error { return nil }

type integrationFakeEventBus struct{}

func (b *integrationFakeEventBus) Publish(_ string, _ ...*message.Message) error { return nil }
func (b *integrationFakeEventBus) Subscribe(_ context.Context, _ string) (<-chan *message.Message, error) {
	return nil, nil
}
func (b *integrationFakeEventBus) Close() error                                   { return nil }
func (b *integrationFakeEventBus) GetNATSConnection() *nats.Conn                  { return nil }
func (b *integrationFakeEventBus) GetJetStream() jetstream.JetStream              { return nil }
func (b *integrationFakeEventBus) GetHealthCheckers() []eventbus.HealthChecker    { return nil }
func (b *integrationFakeEventBus) CreateStream(_ context.Context, _ string) error { return nil }
func (b *integrationFakeEventBus) SubscribeForTest(_ context.Context, _ string) (<-chan *message.Message, error) {
	return nil, nil
}

type integrationFakeHelpers struct{}

func (h *integrationFakeHelpers) CreateResultMessage(_ *message.Message, _ interface{}, _ string) (*message.Message, error) {
	return message.NewMessage("id", nil), nil
}
func (h *integrationFakeHelpers) CreateNewMessage(_ interface{}, _ string) (*message.Message, error) {
	return message.NewMessage("id", nil), nil
}
func (h *integrationFakeHelpers) UnmarshalPayload(_ *message.Message, _ interface{}) error {
	return nil
}

// ─── Test server helper ────────────────────────────────────────────────────────

func newIntegrationServer(svc authservice.Service) *httptest.Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	h := authhandlers.NewAuthHandlers(
		svc,
		&integrationFakeEventBus{},
		&integrationFakeHelpers{},
		logger,
		tracer,
		false,
		"",
		"",
	)

	r := chi.NewRouter()
	// Activity token exchange is a public route (no auth middleware)
	r.Post("/api/auth/discord-activity/token-exchange", h.HandleActivityTokenExchange)
	// Ticket endpoint requires a valid cookie or bearer token
	r.With(authhandlers.AuthMiddleware).Post("/api/auth/ticket", h.HandleHTTPTicket)

	return httptest.NewServer(r)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestActivityTokenExchange_HTTPIntegration_Success(t *testing.T) {
	svc := &integrationFakeService{
		handleOAuthCallbackFn: func(_ context.Context, provider, code, _ string) (*authservice.LoginResponse, error) {
			if provider != "discord-activity" {
				t.Errorf("expected provider 'discord-activity', got %q", provider)
			}
			if code != "test-code" {
				t.Errorf("expected code 'test-code', got %q", code)
			}
			return &authservice.LoginResponse{RefreshToken: "rt-from-exchange", UserUUID: "user-123"}, nil
		},
		getTicketFn: func(_ context.Context, rt, _ string) (*authservice.TicketResponse, error) {
			return &authservice.TicketResponse{NATSToken: "nats-jwt-token", RefreshToken: rt}, nil
		},
	}
	srv := newIntegrationServer(svc)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"code": "test-code"})
	resp, err := http.Post(
		srv.URL+"/api/auth/discord-activity/token-exchange",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON Content-Type, got %q", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["refresh_token"] != "rt-from-exchange" {
		t.Errorf("refresh_token = %q, want 'rt-from-exchange'", result["refresh_token"])
	}
	if result["ticket"] != "nats-jwt-token" {
		t.Errorf("ticket = %q, want 'nats-jwt-token'", result["ticket"])
	}
	if result["user_uuid"] != "user-123" {
		t.Errorf("user_uuid = %q, want 'user-123'", result["user_uuid"])
	}
}

func TestActivityTokenExchange_HTTPIntegration_MissingCode_Returns400(t *testing.T) {
	srv := newIntegrationServer(&integrationFakeService{})
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"code": ""})
	resp, err := http.Post(
		srv.URL+"/api/auth/discord-activity/token-exchange",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestActivityTokenExchange_HTTPIntegration_OAuthFailure_Returns401(t *testing.T) {
	svc := &integrationFakeService{
		handleOAuthCallbackFn: func(_ context.Context, _, _, _ string) (*authservice.LoginResponse, error) {
			return nil, errors.New("discord rejected the code")
		},
	}
	srv := newIntegrationServer(svc)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"code": "bad-code"})
	resp, err := http.Post(
		srv.URL+"/api/auth/discord-activity/token-exchange",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestActivityTokenExchange_HTTPIntegration_GetMethod_Returns405(t *testing.T) {
	srv := newIntegrationServer(&integrationFakeService{})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/auth/discord-activity/token-exchange")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestHTTPTicket_HTTPIntegration_BearerIncludesRefreshTokenInJSON(t *testing.T) {
	validBearer := strings.Repeat("0123456789abcdef", 4) // 64-char hex
	svc := &integrationFakeService{
		getTicketFn: func(_ context.Context, _, _ string) (*authservice.TicketResponse, error) {
			return &authservice.TicketResponse{NATSToken: "nats-jwt", RefreshToken: "new-rt"}, nil
		},
	}
	srv := newIntegrationServer(svc)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+validBearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["refresh_token"] == "" {
		t.Error("expected refresh_token in JSON body for bearer caller")
	}
	if result["ticket"] == "" {
		t.Error("expected ticket in JSON body")
	}
}
