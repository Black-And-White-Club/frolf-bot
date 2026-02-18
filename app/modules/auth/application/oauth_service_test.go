package authservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	authoauth "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/oauth"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// fakeProvider is a minimal OAuth provider stub for testing.
type fakeProvider struct {
	name        string
	exchangeFn  func(ctx context.Context, code string) (*authoauth.UserInfo, error)
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) AuthCodeURL(state string) string {
	return "https://provider.example/auth?state=" + state
}
func (f *fakeProvider) Exchange(ctx context.Context, code string) (*authoauth.UserInfo, error) {
	if f.exchangeFn != nil {
		return f.exchangeFn(ctx, code)
	}
	return &authoauth.UserInfo{
		Provider:    f.name,
		ProviderID:  "user-123",
		DisplayName: "Test User",
	}, nil
}

func newTestRegistry(providers ...authoauth.Provider) *authoauth.Registry {
	r := authoauth.NewRegistry()
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

func newOAuthService(repo *userdb.FakeRepository, registry *authoauth.Registry) Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	return NewService(&FakeJWTProvider{}, &FakeUserJWTBuilder{}, repo, Config{}, logger, tracer, nil, registry)
}

// --- HandleOAuthCallback ---

func TestHandleOAuthCallback_UnknownProvider(t *testing.T) {
	s := newOAuthService(&userdb.FakeRepository{}, newTestRegistry())
	_, err := s.HandleOAuthCallback(context.Background(), "nonexistent", "code", "state")
	if err == nil || err.Error() != "unknown oauth provider: nonexistent" {
		t.Fatalf("expected unknown provider error, got %v", err)
	}
}

func TestHandleOAuthCallback_ExchangeFails(t *testing.T) {
	provider := &fakeProvider{
		name: "discord",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return nil, errors.New("token exchange error")
		},
	}
	s := newOAuthService(&userdb.FakeRepository{}, newTestRegistry(provider))
	_, err := s.HandleOAuthCallback(context.Background(), "discord", "bad-code", "state")
	if err == nil || !errors.Is(err, err) {
		t.Fatalf("expected exchange error, got %v", err)
	}
}

func TestHandleOAuthCallback_Discord_ReturningUser_LinkedIdentityExists(t *testing.T) {
	existingUUID := uuid.New()
	provider := &fakeProvider{name: "discord"}

	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, provider, providerID string) (uuid.UUID, error) {
			return existingUUID, nil // already linked
		},
		UpdateLinkedIdentityTokenFn: func(_ context.Context, _ bun.IDB, _, _, _ string, _ *time.Time) error {
			return nil
		},
		SaveRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ *userdb.RefreshToken) error {
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	resp, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserUUID != existingUUID.String() {
		t.Errorf("expected UUID %s, got %s", existingUUID, resp.UserUUID)
	}
}

func TestHandleOAuthCallback_Discord_BotUserFirstPWALogin_BridgesAccounts(t *testing.T) {
	// The user signed up via Discord bot (has users.user_id) but has no linked_identity.
	// Logging in via Discord OAuth should bridge to their existing account, not create a new one.
	botUserUUID := uuid.New()
	discordSnowflake := "111222333444555666"

	insertLinkedIdentityCalled := false
	createUserCalled := false

	provider := &fakeProvider{
		name: "discord",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return &authoauth.UserInfo{
				Provider:    "discord",
				ProviderID:  discordSnowflake,
				DisplayName: "Jace",
			}, nil
		},
	}

	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound // no linked identity yet
		},
		GetUUIDByDiscordIDFn: func(_ context.Context, _ bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
			if string(discordID) != discordSnowflake {
				t.Errorf("unexpected Discord ID: %s", discordID)
			}
			return botUserUUID, nil // found the bot-created account
		},
		InsertLinkedIdentityFn: func(_ context.Context, _ bun.IDB, userUUID uuid.UUID, provider, providerID, displayName string) error {
			insertLinkedIdentityCalled = true
			if userUUID != botUserUUID {
				t.Errorf("expected bot user UUID %s, got %s", botUserUUID, userUUID)
			}
			if provider != "discord" || providerID != discordSnowflake {
				t.Errorf("unexpected identity: provider=%s providerID=%s", provider, providerID)
			}
			return nil
		},
		CreateUserWithLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _, _ string) (uuid.UUID, error) {
			createUserCalled = true
			return uuid.New(), nil // should NOT be called
		},
		UpdateLinkedIdentityTokenFn: func(_ context.Context, _ bun.IDB, _, _, _ string, _ *time.Time) error {
			return nil
		},
		SaveRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ *userdb.RefreshToken) error {
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	resp, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserUUID != botUserUUID.String() {
		t.Errorf("expected bot user UUID %s, got %s", botUserUUID, resp.UserUUID)
	}
	if !insertLinkedIdentityCalled {
		t.Error("expected InsertLinkedIdentity to be called")
	}
	if createUserCalled {
		t.Error("CreateUserWithLinkedIdentity must NOT be called when bridging a bot account")
	}
}

func TestHandleOAuthCallback_Discord_TrulyNewUser_CreatesAccount(t *testing.T) {
	// No linked identity AND no bot-created account — create fresh.
	newUUID := uuid.New()
	createUserCalled := false

	provider := &fakeProvider{name: "discord"}
	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound
		},
		GetUUIDByDiscordIDFn: func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound // no bot account either
		},
		CreateUserWithLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _, _ string) (uuid.UUID, error) {
			createUserCalled = true
			return newUUID, nil
		},
		UpdateLinkedIdentityTokenFn: func(_ context.Context, _ bun.IDB, _, _, _ string, _ *time.Time) error {
			return nil
		},
		SaveRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ *userdb.RefreshToken) error {
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	resp, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserUUID != newUUID.String() {
		t.Errorf("expected new UUID %s, got %s", newUUID, resp.UserUUID)
	}
	if !createUserCalled {
		t.Error("expected CreateUserWithLinkedIdentity to be called for a brand new user")
	}
}

func TestHandleOAuthCallback_Google_NewUser_NoDiscordFallback(t *testing.T) {
	// Google users never get the Discord bot fallback — they always go straight to create.
	newUUID := uuid.New()
	getUUIDByDiscordIDCalled := false
	createUserCalled := false

	provider := &fakeProvider{
		name: "google",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return &authoauth.UserInfo{Provider: "google", ProviderID: "google-sub-xyz"}, nil
		},
	}
	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound
		},
		GetUUIDByDiscordIDFn: func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			getUUIDByDiscordIDCalled = true
			return uuid.Nil, userdb.ErrNotFound
		},
		CreateUserWithLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _, _ string) (uuid.UUID, error) {
			createUserCalled = true
			return newUUID, nil
		},
		UpdateLinkedIdentityTokenFn: func(_ context.Context, _ bun.IDB, _, _, _ string, _ *time.Time) error {
			return nil
		},
		SaveRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ *userdb.RefreshToken) error {
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	_, err := s.HandleOAuthCallback(context.Background(), "google", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getUUIDByDiscordIDCalled {
		t.Error("GetUUIDByDiscordID must NOT be called for non-discord providers")
	}
	if !createUserCalled {
		t.Error("expected CreateUserWithLinkedIdentity to be called for new Google user")
	}
}

func TestHandleOAuthCallback_Discord_GetUUIDByDiscordID_InfraError(t *testing.T) {
	infraErr := errors.New("db connection lost")
	provider := &fakeProvider{name: "discord"}
	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound
		},
		GetUUIDByDiscordIDFn: func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.Nil, infraErr
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	_, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, infraErr) {
		t.Errorf("expected wrapped infraErr, got %v", err)
	}
}

func TestHandleOAuthCallback_Discord_InsertLinkedIdentity_Fails(t *testing.T) {
	insertErr := errors.New("unique constraint violation")
	provider := &fakeProvider{name: "discord"}
	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound
		},
		GetUUIDByDiscordIDFn: func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.New(), nil // found bot account
		},
		InsertLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _ uuid.UUID, _, _, _ string) error {
			return insertErr
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	_, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err == nil {
		t.Fatal("expected error from InsertLinkedIdentity, got nil")
	}
	if !errors.Is(err, insertErr) {
		t.Errorf("expected wrapped insertErr, got %v", err)
	}
}

func TestHandleOAuthCallback_RefreshToken_IsGenerated(t *testing.T) {
	provider := &fakeProvider{name: "discord"}
	savedToken := (*userdb.RefreshToken)(nil)

	repo := &userdb.FakeRepository{
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.New(), nil
		},
		UpdateLinkedIdentityTokenFn: func(_ context.Context, _ bun.IDB, _, _, _ string, _ *time.Time) error {
			return nil
		},
		SaveRefreshTokenFn: func(_ context.Context, _ bun.IDB, token *userdb.RefreshToken) error {
			savedToken = token
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	resp, err := s.HandleOAuthCallback(context.Background(), "discord", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Error("expected a refresh token in the response")
	}
	if savedToken == nil {
		t.Fatal("expected refresh token to be saved to the repository")
	}
	if savedToken.Revoked {
		t.Error("newly created refresh token must not be revoked")
	}
	if savedToken.ExpiresAt.Before(time.Now()) {
		t.Error("refresh token must expire in the future")
	}
}

// --- LinkIdentityToUser ---

func TestLinkIdentityToUser_Success(t *testing.T) {
	userUUID := uuid.New()
	provider := &fakeProvider{
		name: "google",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return &authoauth.UserInfo{Provider: "google", ProviderID: "google-sub-abc"}, nil
		},
	}

	insertCalled := false
	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return &userdb.RefreshToken{
				UserUUID:  userUUID,
				ExpiresAt: time.Now().Add(time.Hour),
				Revoked:   false,
			}, nil
		},
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return uuid.Nil, userdb.ErrNotFound // not linked yet
		},
		InsertLinkedIdentityFn: func(_ context.Context, _ bun.IDB, u uuid.UUID, _, _, _ string) error {
			insertCalled = true
			if u != userUUID {
				t.Errorf("expected userUUID %s, got %s", userUUID, u)
			}
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "valid-refresh-token", "google", "code", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !insertCalled {
		t.Error("expected InsertLinkedIdentity to be called")
	}
}

func TestLinkIdentityToUser_AlreadyLinkedToSameUser_Idempotent(t *testing.T) {
	userUUID := uuid.New()
	provider := &fakeProvider{
		name: "google",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return &authoauth.UserInfo{Provider: "google", ProviderID: "google-sub-abc"}, nil
		},
	}

	insertCalled := false
	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return &userdb.RefreshToken{
				UserUUID:  userUUID,
				ExpiresAt: time.Now().Add(time.Hour),
				Revoked:   false,
			}, nil
		},
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return userUUID, nil // already linked to THIS user
		},
		InsertLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _ uuid.UUID, _, _, _ string) error {
			insertCalled = true
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "valid-refresh-token", "google", "code", "state")
	if err != nil {
		t.Fatalf("expected idempotent success, got error: %v", err)
	}
	if insertCalled {
		t.Error("InsertLinkedIdentity must NOT be called when identity is already linked to the same user")
	}
}

func TestLinkIdentityToUser_AlreadyLinkedToDifferentUser_Rejected(t *testing.T) {
	// CRITICAL: Provider identity belongs to a different user — must be rejected.
	// This is the primary account-mixing prevention test.
	userUUID := uuid.New()
	differentUserUUID := uuid.New()

	provider := &fakeProvider{
		name: "google",
		exchangeFn: func(_ context.Context, _ string) (*authoauth.UserInfo, error) {
			return &authoauth.UserInfo{Provider: "google", ProviderID: "google-sub-abc"}, nil
		},
	}

	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return &userdb.RefreshToken{
				UserUUID:  userUUID,
				ExpiresAt: time.Now().Add(time.Hour),
				Revoked:   false,
			}, nil
		},
		FindUserByLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _, _ string) (uuid.UUID, error) {
			return differentUserUUID, nil // belongs to someone else
		},
		InsertLinkedIdentityFn: func(_ context.Context, _ bun.IDB, _ uuid.UUID, _, _, _ string) error {
			// Must never reach here.
			t.Error("InsertLinkedIdentity must NOT be called when identity belongs to a different user")
			return nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "valid-refresh-token", "google", "code", "state")
	if err == nil {
		t.Fatal("expected error when linking identity that belongs to a different user, got nil")
	}
}

func TestLinkIdentityToUser_ExpiredSession_Rejected(t *testing.T) {
	provider := &fakeProvider{name: "google"}
	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return &userdb.RefreshToken{
				UserUUID:  uuid.New(),
				ExpiresAt: time.Now().Add(-time.Hour), // expired
				Revoked:   false,
			}, nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "expired-token", "google", "code", "state")
	if err == nil {
		t.Fatal("expected error for expired session, got nil")
	}
}

func TestLinkIdentityToUser_RevokedSession_Rejected(t *testing.T) {
	provider := &fakeProvider{name: "google"}
	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return &userdb.RefreshToken{
				UserUUID:  uuid.New(),
				ExpiresAt: time.Now().Add(time.Hour),
				Revoked:   true, // revoked
			}, nil
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "revoked-token", "google", "code", "state")
	if err == nil {
		t.Fatal("expected error for revoked session, got nil")
	}
}

func TestLinkIdentityToUser_InvalidSession_Rejected(t *testing.T) {
	provider := &fakeProvider{name: "google"}
	repo := &userdb.FakeRepository{
		GetRefreshTokenFn: func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
			return nil, userdb.ErrNotFound
		},
	}

	s := newOAuthService(repo, newTestRegistry(provider))
	err := s.LinkIdentityToUser(context.Background(), "unknown-token", "google", "code", "state")
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
}
