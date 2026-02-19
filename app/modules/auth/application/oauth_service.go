package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// InitiateOAuthLogin generates the provider redirect URL and a random CSRF state token.
// The caller is responsible for storing the state in a short-lived cookie for validation
// in the subsequent callback.
func (s *service) InitiateOAuthLogin(ctx context.Context, providerName string) (redirectURL, state string, err error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.InitiateOAuthLogin")
	defer span.End()

	if s.oauthRegistry == nil {
		return "", "", fmt.Errorf("oauth not configured")
	}

	provider, ok := s.oauthRegistry.Get(providerName)
	if !ok {
		return "", "", fmt.Errorf("unknown oauth provider: %s", providerName)
	}

	// 16 random bytes → 32 hex chars; plenty of entropy for a CSRF state token.
	state, err = generateRandomToken(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	redirectURL = provider.AuthCodeURL(state)

	s.logger.InfoContext(ctx, "Initiated OAuth login", attr.String("provider", providerName))
	return redirectURL, state, nil
}

// HandleOAuthCallback exchanges the authorization code, resolves or creates the canonical
// frolf-bot user, generates a refresh token, and returns the login response.
// CSRF validation (state cookie vs state param) is the caller's responsibility.
func (s *service) HandleOAuthCallback(ctx context.Context, providerName, code, state string) (*LoginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.HandleOAuthCallback")
	defer span.End()

	if s.oauthRegistry == nil {
		return nil, fmt.Errorf("oauth not configured")
	}

	provider, ok := s.oauthRegistry.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("unknown oauth provider: %s", providerName)
	}

	userInfo, err := provider.Exchange(ctx, code)
	if err != nil {
		s.logger.WarnContext(ctx, "OAuth code exchange failed",
			attr.String("provider", providerName),
			attr.Error(err),
		)
		return nil, fmt.Errorf("oauth exchange failed: %w", err)
	}

	var resp *LoginResponse

	err = s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		// Find existing user or create one.
		userUUID, err := s.repo.FindUserByLinkedIdentity(ctx, tx, userInfo.Provider, userInfo.ProviderID)
		if err != nil {
			if !errors.Is(err, userdb.ErrNotFound) {
				return fmt.Errorf("identity lookup failed: %w", err)
			}
			// No linked identity found. For Discord, check if this Discord ID already
			// has an account created by the bot signup flow (users.user_id column).
			if userInfo.Provider == "discord" {
				existingUUID, discordErr := s.repo.GetUUIDByDiscordID(ctx, tx, sharedtypes.DiscordID(userInfo.ProviderID))
				if discordErr == nil {
					// Bot-created account found — bridge it by inserting a linked identity.
					if err = s.repo.InsertLinkedIdentity(ctx, tx, existingUUID, userInfo.Provider, userInfo.ProviderID, userInfo.DisplayName); err != nil {
						return fmt.Errorf("failed to link discord identity to existing user: %w", err)
					}
					userUUID = existingUUID
					s.logger.InfoContext(ctx, "Linked Discord OAuth to existing bot-created user",
						attr.String("provider", providerName),
						attr.String("user_uuid", userUUID.String()),
					)
				} else if !errors.Is(discordErr, userdb.ErrNotFound) {
					return fmt.Errorf("discord id lookup failed: %w", discordErr)
				} else {
					// Truly new user — not in the system via bot or OAuth before.
					userUUID, err = s.repo.CreateUserWithLinkedIdentity(ctx, tx, userInfo.Provider, userInfo.ProviderID, userInfo.DisplayName)
					if err != nil {
						return fmt.Errorf("failed to create user: %w", err)
					}
					s.logger.InfoContext(ctx, "Created new user via OAuth",
						attr.String("provider", providerName),
						attr.String("user_uuid", userUUID.String()),
					)
				}
			} else {
				// Non-discord provider — no legacy fallback, just create.
				userUUID, err = s.repo.CreateUserWithLinkedIdentity(ctx, tx, userInfo.Provider, userInfo.ProviderID, userInfo.DisplayName)
				if err != nil {
					return fmt.Errorf("failed to create user: %w", err)
				}
				s.logger.InfoContext(ctx, "Created new user via OAuth",
					attr.String("provider", providerName),
					attr.String("user_uuid", userUUID.String()),
				)
			}
		}

		// Store the OAuth access token so other services (e.g. club discovery) can call
		// the provider API on behalf of this user. Non-fatal: a token storage failure
		// must not block the login.
		if userInfo.AccessToken != "" {
			if tokenErr := s.repo.UpdateLinkedIdentityToken(ctx, tx, userInfo.Provider, userInfo.ProviderID, userInfo.AccessToken, userInfo.AccessTokenExpiresAt); tokenErr != nil {
				s.logger.WarnContext(ctx, "Failed to store OAuth access token",
					attr.String("provider", providerName),
					attr.Error(tokenErr),
				)
			}
		}

		// Generate refresh token (same pattern as LoginUser).
		rawToken, err := generateRandomToken(32)
		if err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}
		familyID, err := generateRandomToken(16)
		if err != nil {
			return fmt.Errorf("failed to generate token family: %w", err)
		}

		refreshToken := &userdb.RefreshToken{
			Hash:        hashToken(rawToken),
			UserUUID:    userUUID,
			TokenFamily: familyID,
			ExpiresAt:   time.Now().Add(RefreshTokenExpiry),
			Revoked:     false,
		}

		if err := s.repo.SaveRefreshToken(ctx, tx, refreshToken); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}

		resp = &LoginResponse{
			RefreshToken: rawToken,
			UserUUID:     userUUID.String(),
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "OAuth login successful",
		attr.String("provider", providerName),
		attr.String("user_uuid", resp.UserUUID),
	)
	return resp, nil
}

// LinkIdentityToUser links an additional OAuth provider identity to an existing account.
// rawRefreshToken is used to authenticate the caller and look up their user UUID.
// Returns an error if the provider identity is already linked to a different account.
func (s *service) LinkIdentityToUser(ctx context.Context, rawRefreshToken, providerName, code, state string) error {
	ctx, span := s.tracer.Start(ctx, "AuthService.LinkIdentityToUser")
	defer span.End()

	if s.oauthRegistry == nil {
		return fmt.Errorf("oauth not configured")
	}

	// Resolve authenticated user UUID from the refresh token.
	hashedToken := hashToken(rawRefreshToken)
	existingToken, err := s.repo.GetRefreshToken(ctx, nil, hashedToken)
	if err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}
	if existingToken.Revoked || time.Now().After(existingToken.ExpiresAt) {
		return fmt.Errorf("session expired or revoked")
	}
	userUUID := existingToken.UserUUID

	provider, ok := s.oauthRegistry.Get(providerName)
	if !ok {
		return fmt.Errorf("unknown oauth provider: %s", providerName)
	}

	userInfo, err := provider.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("oauth exchange failed: %w", err)
	}

	// Check whether this provider identity is already linked to any account.
	existingUUID, err := s.repo.FindUserByLinkedIdentity(ctx, nil, userInfo.Provider, userInfo.ProviderID)
	if err != nil && !errors.Is(err, userdb.ErrNotFound) {
		return fmt.Errorf("identity lookup failed: %w", err)
	}
	if err == nil {
		// Found an existing link.
		if existingUUID != userUUID {
			return fmt.Errorf("this %s account is already linked to a different frolf-bot user", providerName)
		}
		// Already linked to the same user — idempotent success.
		s.logger.InfoContext(ctx, "Identity already linked to user (no-op)",
			attr.String("provider", providerName),
			attr.String("user_uuid", userUUID.String()),
		)

		// Update the access token to ensure we have the latest credentials/scopes
		if userInfo.AccessToken != "" {
			if err := s.repo.UpdateLinkedIdentityToken(ctx, nil, userInfo.Provider, userInfo.ProviderID, userInfo.AccessToken, userInfo.AccessTokenExpiresAt); err != nil {
				s.logger.WarnContext(ctx, "Failed to update OAuth access token on re-link", attr.Error(err))
			}
		}

		return nil
	}

	// Insert the new linked identity.
	if err := s.repo.InsertLinkedIdentity(ctx, nil, userUUID, userInfo.Provider, userInfo.ProviderID, userInfo.DisplayName); err != nil {
		return fmt.Errorf("failed to link identity: %w", err)
	}

	s.logger.InfoContext(ctx, "Linked new identity to user",
		attr.String("provider", providerName),
		attr.String("user_uuid", userUUID.String()),
	)
	return nil
}

// UnlinkProvider removes an OAuth provider identity from the user's account.
// The user must have at least one other linked identity remaining.
func (s *service) UnlinkProvider(ctx context.Context, rawRefreshToken, providerName string) error {
	ctx, span := s.tracer.Start(ctx, "AuthService.UnlinkProvider")
	defer span.End()

	userUUID, err := s.getUserUUIDFromRefreshToken(ctx, rawRefreshToken)
	if err != nil {
		return err
	}

	providers, err := s.repo.GetLinkedProvidersByUserUUID(ctx, nil, userUUID)
	if err != nil {
		return fmt.Errorf("failed to fetch linked providers: %w", err)
	}

	linked := false
	for _, p := range providers {
		if p == providerName {
			linked = true
			break
		}
	}
	if !linked {
		return fmt.Errorf("provider %s is not linked to this account", providerName)
	}

	if err := s.repo.DeleteLinkedIdentity(ctx, nil, userUUID, providerName); err != nil {
		return fmt.Errorf("failed to unlink provider: %w", err)
	}

	s.logger.InfoContext(ctx, "Unlinked provider from user",
		attr.String("provider", providerName),
		attr.String("user_uuid", userUUID.String()),
	)
	return nil
}

// getUserUUIDFromRefreshToken is a private helper used by other oauth service methods.
// It validates the refresh token and returns the associated user UUID.
func (s *service) getUserUUIDFromRefreshToken(ctx context.Context, rawToken string) (uuid.UUID, error) {
	hash := hashToken(rawToken)
	token, err := s.repo.GetRefreshToken(ctx, nil, hash)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	if token.Revoked || time.Now().After(token.ExpiresAt) {
		return uuid.Nil, fmt.Errorf("session expired or revoked")
	}
	return token.UserUUID, nil
}
