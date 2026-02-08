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
	"github.com/uptrace/bun"
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
// The entire operation runs in a transaction to prevent double-redemption of magic links.
func (s *service) LoginUser(ctx context.Context, oneTimeToken string) (*LoginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.LoginUser")
	defer span.End()

	var resp *LoginResponse

	err := s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		// 1. Validate the one-time token (DB)
		ml, err := s.repo.GetMagicLink(ctx, tx, oneTimeToken)
		if err != nil {
			s.logger.WarnContext(ctx, "Invalid one-time token", attr.Error(err))
			return fmt.Errorf("invalid token: %w", err)
		}

		if ml.Used {
			s.logger.WarnContext(ctx, "One-time token already used", attr.String("token", oneTimeToken))
			return fmt.Errorf("token already used")
		}

		if time.Now().After(ml.ExpiresAt) {
			s.logger.WarnContext(ctx, "One-time token expired", attr.String("token", oneTimeToken))
			return fmt.Errorf("token expired")
		}

		// Mark as used
		if err := s.repo.MarkMagicLinkUsed(ctx, tx, oneTimeToken); err != nil {
			s.logger.ErrorContext(ctx, "Failed to mark token used", attr.Error(err))
			return fmt.Errorf("failed to process token: %w", err)
		}

		userUUID := ml.UserUUID

		// 2. Generate a refresh token
		token, err := generateRandomToken(32)
		if err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}

		hashedToken := hashToken(token)
		familyID, err := generateRandomToken(16)
		if err != nil {
			return fmt.Errorf("failed to generate token family: %w", err)
		}

		refreshToken := &userdb.RefreshToken{
			Hash:        hashedToken,
			UserUUID:    userUUID,
			TokenFamily: familyID,
			ExpiresAt:   time.Now().Add(RefreshTokenExpiry),
			Revoked:     false,
		}

		// Save to DB
		if err := s.repo.SaveRefreshToken(ctx, tx, refreshToken); err != nil {
			s.logger.ErrorContext(ctx, "Failed to save refresh token",
				attr.Error(err),
				attr.String("user_uuid", userUUID.String()),
			)
			return fmt.Errorf("failed to save session: %w", err)
		}

		resp = &LoginResponse{
			RefreshToken: token,
			UserUUID:     userUUID.String(),
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTicket validates a refresh token and mints a short-lived NATS ticket.
func (s *service) GetTicket(ctx context.Context, rawToken string, activeClubUUID string) (*TicketResponse, error) {
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
		if err := s.repo.RevokeAllUserTokens(ctx, nil, token.UserUUID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to revoke token family",
				attr.Error(err),
				attr.String("user_uuid", token.UserUUID.String()),
			)
		}
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

	var activeUUID uuid.UUID
	var activeRole authdomain.Role = authdomain.RolePlayer
	var clubs []authdomain.ClubRole
	var syncRequests []SyncRequest

	if len(memberships) == 0 {
		memberships = s.backfillClubMemberships(ctx, token.UserUUID)
	}

	if len(memberships) > 0 {
		// Default to first membership
		activeUUID = memberships[0].ClubUUID
		activeRole = authdomain.Role(memberships[0].Role)

		// If a specific active club was requested, try to find it
		if activeClubUUID != "" {
			if targetUUID, err := uuid.Parse(activeClubUUID); err == nil {
				found := false
				for _, m := range memberships {
					if m.ClubUUID == targetUUID {
						activeUUID = m.ClubUUID
						activeRole = authdomain.Role(m.Role)
						found = true
						break
					}
				}
				if !found {
					s.logger.WarnContext(ctx, "Requested active club not found in user memberships",
						attr.String("requested_uuid", activeClubUUID),
						attr.String("user_uuid", token.UserUUID.String()),
					)
				}
			}
		}

		// Get global user for fallback display name
		var globalDisplayName string
		user, err := s.repo.GetUserByUUID(ctx, nil, token.UserUUID)
		if err == nil && user != nil {
			globalDisplayName = user.GetDisplayName()
		}

		for _, m := range memberships {
			// Request async profile sync if:
			// 1. display_name is NULL (never synced), OR
			// 2. synced_at is NULL or older than ProfileSyncStaleness (stale data)
			needsSync := m.DisplayName == nil ||
				m.SyncedAt == nil ||
				time.Since(*m.SyncedAt) > ProfileSyncStaleness

			if needsSync && m.ExternalID != nil {
				if gid, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, nil, m.ClubUUID); err == nil {
					syncRequests = append(syncRequests, SyncRequest{
						UserID:  *m.ExternalID,
						GuildID: string(gid),
					})
				}
			}

			clubs = append(clubs, authdomain.ClubRole{
				ClubUUID:    m.ClubUUID,
				Role:        authdomain.Role(m.Role),
				DisplayName: resolveDisplayName(m.DisplayName, globalDisplayName),
			})
		}
	}

	// Resolve Discord Guild ID from active club
	// This is critical for NATS permissions if the frontend subscribes using Guild ID
	var guildID string
	if activeUUID != uuid.Nil {
		if gid, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, nil, activeUUID); err == nil {
			guildID = string(gid)
		} else {
			s.logger.WarnContext(ctx, "Failed to resolve Discord Guild ID for club",
				attr.String("club_uuid", activeUUID.String()),
				attr.Error(err),
			)
			// Fallback to club UUID string if resolution fails, though this may not match frontend expectations
			guildID = activeUUID.String()
		}
	}

	// 4. Mint NATS Ticket with the hash of the NEW rotated refresh token
	claims := &authdomain.Claims{
		UserID:           token.UserUUID.String(), // Use UUID as ID for now since we are decoupling
		UserUUID:         token.UserUUID,
		GuildID:          guildID,
		ActiveClubUUID:   activeUUID,
		Role:             activeRole,
		Clubs:            clubs,
		RefreshTokenHash: newHashed,
	}

	// 4. Mint standard HMAC JWT as a "ticket"
	// This ticket will be exchanged for a NATS User JWT via Auth Callout
	natsToken, err := s.jwtProvider.GenerateToken(claims, TicketTTL)
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

	err = s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		if err := s.repo.SaveRefreshToken(ctx, tx, newRefreshToken); err != nil {
			return fmt.Errorf("failed to save rotated token: %w", err)
		}
		if err := s.repo.RevokeRefreshToken(ctx, tx, hashedToken); err != nil {
			return fmt.Errorf("failed to revoke old token: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &TicketResponse{
		NATSToken:    natsToken,
		RefreshToken: newToken,
		SyncRequests: syncRequests,
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

// backfillClubMemberships checks for legacy guild memberships and creates
// corresponding club memberships if the user has none yet.
func (s *service) backfillClubMemberships(ctx context.Context, userUUID uuid.UUID) []*userdb.ClubMembership {
	user, err := s.repo.GetUserByUUID(ctx, nil, userUUID)
	if err != nil || user == nil || user.UserID == nil {
		return nil
	}

	legacyMemberships, err := s.repo.GetUserMemberships(ctx, nil, *user.UserID)
	if err != nil || len(legacyMemberships) == 0 {
		return nil
	}

	s.logger.InfoContext(ctx, "Backfilling club memberships for user", attr.String("user_uuid", userUUID.String()))
	for _, lm := range legacyMemberships {
		clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, nil, lm.GuildID)
		if err != nil {
			continue
		}
		extID := string(lm.UserID)
		emptyName := ""
		cm := &userdb.ClubMembership{
			ClubUUID:    clubUUID,
			UserUUID:    userUUID,
			Role:        lm.Role,
			JoinedAt:    lm.JoinedAt,
			ExternalID:  &extID,
			DisplayName: &emptyName,
		}
		if err := s.repo.UpsertClubMembership(ctx, nil, cm); err != nil {
			s.logger.WarnContext(ctx, "Failed to backfill club membership", attr.Error(err))
		}
	}

	// Re-fetch memberships after backfill
	memberships, _ := s.repo.GetClubMembershipsByUserUUID(ctx, nil, userUUID)
	return memberships
}

// resolveDisplayName returns the club display name if present, otherwise fallback.
func resolveDisplayName(clubDisplayName *string, fallback string) string {
	if clubDisplayName != nil && *clubDisplayName != "" {
		return *clubDisplayName
	}
	return fallback
}
