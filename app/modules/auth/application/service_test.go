package authservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestService_GenerateMagicLink(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{
		PWABaseURL: "https://frolf.bot",
		DefaultTTL: 1 * time.Hour,
	}

	t.Run("success", func(t *testing.T) {
		fakeJWT := "valid-jwt"
		jwtProvider := &FakeJWTProvider{
			GenerateTokenFunc: func(userID, guildID string, role authdomain.Role, ttl time.Duration) (string, error) {
				return fakeJWT, nil
			},
		}
		s := NewService(jwtProvider, &FakeUserJWTBuilder{}, config, logger, tracer)

		resp, err := s.GenerateMagicLink(ctx, "u1", "g1", authdomain.RolePlayer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got failure with error: %s", resp.Error)
		}

		expectedURL := "https://frolf.bot?t=" + fakeJWT
		if resp.URL != expectedURL {
			t.Errorf("expected URL %s, got %s", expectedURL, resp.URL)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		s := NewService(&FakeJWTProvider{}, &FakeUserJWTBuilder{}, config, logger, tracer)

		resp, err := s.GenerateMagicLink(ctx, "u1", "g1", authdomain.Role("invalid"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Success {
			t.Error("expected failure for invalid role")
		}

		if resp.Error != ErrInvalidRole.Error() {
			t.Errorf("expected error %v, got %s", ErrInvalidRole, resp.Error)
		}
	})

	t.Run("jwt generation failure", func(t *testing.T) {
		jwtProvider := &FakeJWTProvider{
			GenerateTokenFunc: func(userID, guildID string, role authdomain.Role, ttl time.Duration) (string, error) {
				return "", errors.New("jwt error")
			},
		}
		s := NewService(jwtProvider, &FakeUserJWTBuilder{}, config, logger, tracer)

		resp, err := s.GenerateMagicLink(ctx, "u1", "g1", authdomain.RolePlayer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Success {
			t.Error("expected failure")
		}

		if resp.Error != ErrGenerateToken.Error() {
			t.Errorf("expected error %v, got %s", ErrGenerateToken, resp.Error)
		}
	})
}

func TestService_HandleNATSAuthRequest(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{}

	t.Run("success", func(t *testing.T) {
		jwtProvider := &FakeJWTProvider{
			ValidateTokenFunc: func(tokenString string) (*authdomain.Claims, error) {
				return &authdomain.Claims{UserID: "u1", GuildID: "g1", Role: authdomain.RolePlayer}, nil
			},
		}
		natsBuilder := &FakeUserJWTBuilder{
			BuildUserJWTFunc: func(userID, guildID string, perms *permissions.Permissions) (string, error) {
				return "nats-jwt", nil
			},
		}
		s := NewService(jwtProvider, natsBuilder, config, logger, tracer)

		req := &NATSAuthRequest{
			ConnectOpts: ConnectOptions{Password: "valid-token"},
		}
		resp, err := s.HandleNATSAuthRequest(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Error != "" {
			t.Errorf("unexpected error in response: %s", resp.Error)
		}

		if resp.Jwt != "nats-jwt" {
			t.Errorf("expected nats-jwt, got %s", resp.Jwt)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		s := NewService(&FakeJWTProvider{}, &FakeUserJWTBuilder{}, config, logger, tracer)

		req := &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: ""}}
		resp, err := s.HandleNATSAuthRequest(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Error != ErrMissingToken.Error() {
			t.Errorf("expected error %v, got %s", ErrMissingToken, resp.Error)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		jwtProvider := &FakeJWTProvider{
			ValidateTokenFunc: func(tokenString string) (*authdomain.Claims, error) {
				return nil, errors.New("invalid")
			},
		}
		s := NewService(jwtProvider, &FakeUserJWTBuilder{}, config, logger, tracer)

		req := &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: "bad-token"}}
		resp, err := s.HandleNATSAuthRequest(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(resp.Error, "invalid token") {
			t.Errorf("expected invalid token error, got %s", resp.Error)
		}
	})

	t.Run("nats builder error", func(t *testing.T) {
		jwtProvider := &FakeJWTProvider{
			ValidateTokenFunc: func(tokenString string) (*authdomain.Claims, error) {
				return &authdomain.Claims{UserID: "u1", GuildID: "g1", Role: authdomain.RolePlayer}, nil
			},
		}
		natsBuilder := &FakeUserJWTBuilder{
			BuildUserJWTFunc: func(userID, guildID string, perms *permissions.Permissions) (string, error) {
				return "", errors.New("nats error")
			},
		}
		s := NewService(jwtProvider, natsBuilder, config, logger, tracer)

		req := &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: "valid-token"}}
		resp, err := s.HandleNATSAuthRequest(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Error != ErrGenerateUserJWT.Error() {
			t.Errorf("expected error %v, got %s", ErrGenerateUserJWT, resp.Error)
		}
	})
}
