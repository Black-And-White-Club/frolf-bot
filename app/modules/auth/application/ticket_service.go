package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
)

// TicketService handles ticket vending and session management.

// generateRandomToken creates a secure random token.
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken hashes a token for secure storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// LoginUser validates a one-time token and creates a long-lived session (refresh token).
func (s *service) LoginUser(ctx context.Context, oneTimeToken string) (*LoginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.LoginUser")
	defer span.End()

	// 1. Validate the one-time token (DB)
	ml, err := s.repo.GetMagicLink(ctx, nil, oneTimeToken)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid one-time token", attr.Error(err))
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if ml.Used {
		s.logger.WarnContext(ctx, "One-time token already used", attr.String("token", oneTimeToken))
		return nil, fmt.Errorf("token already used")
	}

	if time.Now().After(ml.ExpiresAt) {
		s.logger.WarnContext(ctx, "One-time token expired", attr.String("token", oneTimeToken))
		return nil, fmt.Errorf("token expired")
	}

	// Mark as used
	if err := s.repo.MarkMagicLinkUsed(ctx, nil, oneTimeToken); err != nil {
		s.logger.ErrorContext(ctx, "Failed to mark token used", attr.Error(err))
		return nil, fmt.Errorf("failed to process token: %w", err)
	}

	userUUID := ml.UserUUID

	// 2. Generate a refresh token
	token, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	hashedToken := hashToken(token)
	familyID, err := generateRandomToken(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token family: %w", err)
	}

	refreshToken := &userdb.RefreshToken{
		Hash:        hashedToken,
		UserUUID:    userUUID,
		TokenFamily: familyID,
		ExpiresAt:   time.Now().Add(RefreshTokenExpiry),
		Revoked:     false,
	}

	// Save to DB
	if err := s.repo.SaveRefreshToken(ctx, nil, refreshToken); err != nil {
		s.logger.ErrorContext(ctx, "Failed to save refresh token",
			attr.Error(err),
			attr.String("user_uuid", userUUID.String()),
		)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &LoginResponse{
		RefreshToken: token,
		UserUUID:     userUUID.String(),
	}, nil
}

// GetTicket validates a refresh token and mints a short-lived NATS ticket.
func (s *service) GetTicket(ctx context.Context, rawToken string) (*TicketResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.GetTicket")
	defer span.End()

	hashedToken := hashToken(rawToken)

	// 1. Retrieve and validate the refresh token
	token, err := s.repo.GetRefreshToken(ctx, nil, hashedToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if token.Revoked {
		s.logger.WarnContext(ctx, "Revoked token used", attr.String("token_hash", hashedToken))
		// Security hardening: Revoke all tokens in family if a revoked token is reused
		_ = s.repo.RevokeAllUserTokens(ctx, nil, token.UserUUID)
		return nil, fmt.Errorf("session revoked")
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// 2. Generate rotated token early so we can put its hash in the claims
	newToken, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rotated token: %w", err)
	}
	newHashed := hashToken(newToken)

	// 3. Load user roles/memberships
	memberships, err := s.repo.GetClubMembershipsByUserUUID(ctx, nil, token.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load memberships: %w", err)
	}

	var activeClubUUID uuid.UUID
	var activeRole authdomain.Role = authdomain.RolePlayer
	var clubs []authdomain.ClubRole

	if len(memberships) > 0 {
		activeClubUUID = memberships[0].ClubUUID
		activeRole = authdomain.Role(memberships[0].Role)
		for _, m := range memberships {
			clubs = append(clubs, authdomain.ClubRole{
				ClubUUID: m.ClubUUID,
				Role:     authdomain.Role(m.Role),
			})
		}
	}

	// 4. Mint NATS Ticket with the hash of the NEW rotated refresh token
	claims := &authdomain.Claims{
		UserUUID:         token.UserUUID,
		ActiveClubUUID:   activeClubUUID,
		Role:             activeRole,
		Clubs:            clubs,
		RefreshTokenHash: newHashed,
	}

	perms := s.permissionBuilder.ForRole(claims)
	if s.userJWTBuilder == nil {
		s.logger.ErrorContext(ctx, "NATS JWT builder not configured (AuthCallout.Enabled likely false)")
		return nil, ErrGenerateUserJWT
	}
	natsToken, err := s.userJWTBuilder.BuildUserJWT(claims, perms)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ticket: %w", err)
	}

	// 5. Finalize Token Rotation (Save new, revoke old)
	newRefreshToken := &userdb.RefreshToken{
		Hash:        newHashed,
		UserUUID:    token.UserUUID,
		TokenFamily: token.TokenFamily,
		ExpiresAt:   time.Now().Add(RefreshTokenExpiry),
		Revoked:     false,
	}

	if err := s.repo.SaveRefreshToken(ctx, nil, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to save rotated token: %w", err)
	}
	_ = s.repo.RevokeRefreshToken(ctx, nil, hashedToken)

	return &TicketResponse{
		NATSToken:    natsToken,
		RefreshToken: newToken,
	}, nil
}

// LogoutUser revokes the refresh token.
func (s *service) LogoutUser(ctx context.Context, rawToken string) error {
	ctx, span := s.tracer.Start(ctx, "AuthService.LogoutUser")
	defer span.End()

	if rawToken == "" {
		return nil
	}

	hashedToken := hashToken(rawToken)

	if err := s.repo.RevokeRefreshToken(ctx, nil, hashedToken); err != nil {
		s.logger.ErrorContext(ctx, "Failed to revoke token", attr.Error(err))
		return err
	}

	return nil
}
