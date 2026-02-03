package authservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
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

	tests := []struct {
		name      string
		userID    string
		guildID   string
		role      authdomain.Role
		setupMock func(j *FakeJWTProvider, n *FakeUserJWTBuilder, r *userdb.FakeRepository)
		verify    func(t *testing.T, resp *MagicLinkResponse, err error)
	}{
		{
			name:    "success",
			userID:  "u1",
			guildID: "g1",
			role:    authdomain.RolePlayer,
			setupMock: func(j *FakeJWTProvider, n *FakeUserJWTBuilder, r *userdb.FakeRepository) {
				userUUID := uuid.New()
				clubUUID := uuid.New()
				r.GetUUIDByDiscordIDFn = func(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
					return userUUID, nil
				}
				r.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				r.GetClubMembershipFn = func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
					return &userdb.ClubMembership{ClubUUID: clubUUID, Role: "player"}, nil
				}
				r.SaveMagicLinkFn = func(ctx context.Context, db bun.IDB, link *userdb.MagicLink) error {
					return nil
				}
			},
			verify: func(t *testing.T, resp *MagicLinkResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !resp.Success {
					t.Errorf("expected success, got failure: %s", resp.Error)
				}
				if !strings.HasPrefix(resp.URL, "https://frolf.bot?t=") {
					t.Errorf("expected URL to start with base URL, got %s", resp.URL)
				}
			},
		},
		{
			name:    "invalid role",
			userID:  "u1",
			guildID: "g1",
			role:    authdomain.Role("invalid"),
			verify: func(t *testing.T, resp *MagicLinkResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Success {
					t.Error("expected failure for invalid role")
				}
				if resp.Error != ErrInvalidRole.Error() {
					t.Errorf("expected error %v, got %s", ErrInvalidRole, resp.Error)
				}
			},
		},
		{
			name:    "save magic link failure",
			userID:  "u1",
			guildID: "g1",
			role:    authdomain.RolePlayer,
			setupMock: func(j *FakeJWTProvider, n *FakeUserJWTBuilder, r *userdb.FakeRepository) {
				userUUID := uuid.New()
				clubUUID := uuid.New()
				r.GetUUIDByDiscordIDFn = func(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
					return userUUID, nil
				}
				r.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				r.GetClubMembershipFn = func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
					return &userdb.ClubMembership{ClubUUID: clubUUID, Role: "player"}, nil
				}
				r.SaveMagicLinkFn = func(ctx context.Context, db bun.IDB, link *userdb.MagicLink) error {
					return errors.New("db error")
				}
			},
			verify: func(t *testing.T, resp *MagicLinkResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Success {
					t.Error("expected failure for save error")
				}
				if resp.Error != "failed to generate magic link" {
					t.Errorf("expected error 'failed to generate magic link', got %s", resp.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtProvider := &FakeJWTProvider{}
			natsBuilder := &FakeUserJWTBuilder{}
			repo := &userdb.FakeRepository{}
			if tt.setupMock != nil {
				tt.setupMock(jwtProvider, natsBuilder, repo)
			}

			s := NewService(jwtProvider, natsBuilder, repo, config, logger, tracer)
			resp, err := s.GenerateMagicLink(ctx, tt.userID, tt.guildID, tt.role)
			tt.verify(t, resp, err)
		})
	}
}

func TestService_ValidateToken(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{}

	tests := []struct {
		name        string
		tokenString string
		setupMock   func(j *FakeJWTProvider)
		verify      func(t *testing.T, claims *authdomain.Claims, err error)
	}{
		{
			name:        "empty token",
			tokenString: "",
			verify: func(t *testing.T, claims *authdomain.Claims, err error) {
				if !errors.Is(err, ErrMissingToken) {
					t.Errorf("expected ErrMissingToken, got %v", err)
				}
			},
		},
		{
			name:        "valid token",
			tokenString: "valid-token",
			setupMock: func(j *FakeJWTProvider) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return &authdomain.Claims{UserID: "u1"}, nil
				}
			},
			verify: func(t *testing.T, claims *authdomain.Claims, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if claims.UserID != "u1" {
					t.Errorf("expected user ID u1, got %s", claims.UserID)
				}
			},
		},
		{
			name:        "invalid token",
			tokenString: "invalid-token",
			setupMock: func(j *FakeJWTProvider) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return nil, errors.New("invalid token")
				}
			},
			verify: func(t *testing.T, claims *authdomain.Claims, err error) {
				if err == nil || !strings.Contains(err.Error(), "invalid token") {
					t.Errorf("expected invalid token error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtProvider := &FakeJWTProvider{}
			if tt.setupMock != nil {
				tt.setupMock(jwtProvider)
			}

			s := NewService(jwtProvider, &FakeUserJWTBuilder{}, &userdb.FakeRepository{}, config, logger, tracer)
			claims, err := s.ValidateToken(ctx, tt.tokenString)
			tt.verify(t, claims, err)
		})
	}
}

func TestService_HandleNATSAuthRequest(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{}

	tests := []struct {
		name      string
		req       *NATSAuthRequest
		setupMock func(j *FakeJWTProvider, n *FakeUserJWTBuilder)
		verify    func(t *testing.T, resp *NATSAuthResponse, err error)
	}{
		{
			name: "success",
			req:  &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: "valid-token"}},
			setupMock: func(j *FakeJWTProvider, n *FakeUserJWTBuilder) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return &authdomain.Claims{UserID: "u1", GuildID: "g1", Role: authdomain.RolePlayer}, nil
				}
				n.BuildUserJWTFunc = func(claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
					return "nats-jwt", nil
				}
			},
			verify: func(t *testing.T, resp *NATSAuthResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Error != "" {
					t.Errorf("unexpected error in response: %s", resp.Error)
				}
				if resp.Jwt != "nats-jwt" {
					t.Errorf("expected nats-jwt, got %s", resp.Jwt)
				}
			},
		},
		{
			name: "missing token",
			req:  &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: ""}},
			verify: func(t *testing.T, resp *NATSAuthResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Error != ErrMissingToken.Error() {
					t.Errorf("expected error %v, got %s", ErrMissingToken, resp.Error)
				}
			},
		},
		{
			name: "invalid token",
			req:  &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: "bad-token"}},
			setupMock: func(j *FakeJWTProvider, n *FakeUserJWTBuilder) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return nil, errors.New("invalid")
				}
			},
			verify: func(t *testing.T, resp *NATSAuthResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(resp.Error, "invalid token") {
					t.Errorf("expected invalid token error, got %s", resp.Error)
				}
			},
		},
		{
			name: "nats builder error",
			req:  &NATSAuthRequest{ConnectOpts: ConnectOptions{Password: "valid-token"}},
			setupMock: func(j *FakeJWTProvider, n *FakeUserJWTBuilder) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return &authdomain.Claims{UserID: "u1", GuildID: "g1", Role: authdomain.RolePlayer}, nil
				}
				n.BuildUserJWTFunc = func(claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
					return "", errors.New("nats error")
				}
			},
			verify: func(t *testing.T, resp *NATSAuthResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Error != ErrGenerateUserJWT.Error() {
					t.Errorf("expected error %v, got %s", ErrGenerateUserJWT, resp.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtProvider := &FakeJWTProvider{}
			natsBuilder := &FakeUserJWTBuilder{}
			if tt.setupMock != nil {
				tt.setupMock(jwtProvider, natsBuilder)
			}

			s := NewService(jwtProvider, natsBuilder, &userdb.FakeRepository{}, config, logger, tracer)
			resp, err := s.HandleNATSAuthRequest(ctx, tt.req)
			tt.verify(t, resp, err)
		})
	}
}

func TestService_LoginUser(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{}
	userUUID := uuid.New()

	tests := []struct {
		name         string
		oneTimeToken string
		setupMock    func(j *FakeJWTProvider, r *userdb.FakeRepository)
		verify       func(t *testing.T, resp *LoginResponse, err error)
	}{
		{
			name:         "success",
			oneTimeToken: "otp",
			setupMock: func(j *FakeJWTProvider, r *userdb.FakeRepository) {
				j.ValidateTokenFunc = func(tokenString string) (*authdomain.Claims, error) {
					return &authdomain.Claims{UserUUID: userUUID}, nil
				}
				r.GetMagicLinkFn = func(ctx context.Context, db bun.IDB, token string) (*userdb.MagicLink, error) {
					return &userdb.MagicLink{
						Token:     token,
						UserUUID:  userUUID,
						ExpiresAt: time.Now().Add(time.Hour),
						Used:      false,
					}, nil
				}
				r.MarkMagicLinkUsedFn = func(ctx context.Context, db bun.IDB, token string) error {
					return nil
				}
				r.SaveRefreshTokenFn = func(ctx context.Context, db bun.IDB, token *userdb.RefreshToken) error {
					return nil
				}
			},
			verify: func(t *testing.T, resp *LoginResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.UserUUID != userUUID.String() {
					t.Errorf("expected user uuid %s, got %s", userUUID.String(), resp.UserUUID)
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh token, got empty")
				}
			},
		},
		{
			name:         "invalid token",
			oneTimeToken: "bad-otp",
			setupMock: func(j *FakeJWTProvider, r *userdb.FakeRepository) {
				r.GetMagicLinkFn = func(ctx context.Context, db bun.IDB, token string) (*userdb.MagicLink, error) {
					return nil, userdb.ErrNotFound
				}
			},
			verify: func(t *testing.T, resp *LoginResponse, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtProvider := &FakeJWTProvider{}
			repo := &userdb.FakeRepository{}
			if tt.setupMock != nil {
				tt.setupMock(jwtProvider, repo)
			}

			s := NewService(jwtProvider, &FakeUserJWTBuilder{}, repo, config, logger, tracer)
			resp, err := s.LoginUser(ctx, tt.oneTimeToken)
			tt.verify(t, resp, err)
		})
	}
}

func TestService_GetTicket(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	config := Config{}
	userUUID := uuid.New()
	clubUUID := uuid.New()

	tests := []struct {
		name         string
		refreshToken string
		setupMock    func(r *userdb.FakeRepository, n *FakeUserJWTBuilder)
		verify       func(t *testing.T, resp *TicketResponse, err error)
	}{
		{
			name:         "success",
			refreshToken: "valid-rt",
			setupMock: func(r *userdb.FakeRepository, n *FakeUserJWTBuilder) {
				r.GetRefreshTokenFn = func(ctx context.Context, db bun.IDB, hash string) (*userdb.RefreshToken, error) {
					return &userdb.RefreshToken{
						UserUUID:    userUUID,
						TokenFamily: "family1",
						ExpiresAt:   time.Now().Add(1 * time.Hour),
						Revoked:     false,
					}, nil
				}
				r.GetClubMembershipsByUserUUIDFn = func(ctx context.Context, db bun.IDB, u uuid.UUID) ([]*userdb.ClubMembership, error) {
					return []*userdb.ClubMembership{{ClubUUID: clubUUID, Role: "admin"}}, nil
				}
				n.BuildUserJWTFunc = func(claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
					return "nats-ticket", nil
				}
			},
			verify: func(t *testing.T, resp *TicketResponse, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.NATSToken != "nats-ticket" {
					t.Errorf("expected nats-ticket, got %s", resp.NATSToken)
				}
			},
		},
		{
			name:         "revoked token",
			refreshToken: "revoked-rt",
			setupMock: func(r *userdb.FakeRepository, n *FakeUserJWTBuilder) {
				r.GetRefreshTokenFn = func(ctx context.Context, db bun.IDB, hash string) (*userdb.RefreshToken, error) {
					return &userdb.RefreshToken{UserUUID: userUUID, Revoked: true}, nil
				}
			},
			verify: func(t *testing.T, resp *TicketResponse, err error) {
				if err == nil || !strings.Contains(err.Error(), "session revoked") {
					t.Errorf("expected session revoked error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userdb.FakeRepository{}
			natsBuilder := &FakeUserJWTBuilder{}
			if tt.setupMock != nil {
				tt.setupMock(repo, natsBuilder)
			}

			s := NewService(&FakeJWTProvider{}, natsBuilder, repo, config, logger, tracer)
			resp, err := s.GetTicket(ctx, tt.refreshToken)
			tt.verify(t, resp, err)
		})
	}
}

func TestService_LogoutUser(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name         string
		refreshToken string
		setupMock    func(r *userdb.FakeRepository)
		verify       func(t *testing.T, err error)
	}{
		{
			name:         "success",
			refreshToken: "rt",
			setupMock: func(r *userdb.FakeRepository) {
				r.RevokeRefreshTokenFn = func(ctx context.Context, db bun.IDB, hash string) error { return nil }
			},
			verify: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userdb.FakeRepository{}
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			s := NewService(&FakeJWTProvider{}, &FakeUserJWTBuilder{}, repo, Config{}, logger, tracer)
			err := s.LogoutUser(ctx, tt.refreshToken)
			tt.verify(t, err)
		})
	}
}
