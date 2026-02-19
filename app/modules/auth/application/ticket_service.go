package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

func tokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:6])
}

// LoginUser validates a one-time token and creates a long-lived session (refresh token).
// The entire operation runs in a transaction to prevent double-redemption of magic links.
func (s *service) LoginUser(ctx context.Context, oneTimeToken string) (*LoginResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.LoginUser")
	defer span.End()

	var resp *LoginResponse

	err := s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		// 1. Atomically consume one-time token.
		now := time.Now().UTC()
		tokenHash := hashToken(oneTimeToken)
		ml, err := s.repo.ConsumeMagicLink(ctx, tx, tokenHash, now)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to consume one-time token",
				attr.Error(err),
				attr.String("token_fingerprint", tokenFingerprint(oneTimeToken)),
			)
			return fmt.Errorf("invalid or expired token")
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
	var replayedUserUUID uuid.UUID

	var resp *TicketResponse
	err := s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		// 1. Retrieve and lock refresh token row.
		token, err := s.repo.GetRefreshTokenForUpdate(ctx, tx, hashedToken)
		if err != nil {
			return fmt.Errorf("invalid refresh token: %w", err)
		}

		if token.Revoked {
			s.logger.WarnContext(ctx, "Revoked token used", attr.String("token_hash", hashedToken))
			replayedUserUUID = token.UserUUID
			return ErrRevokedSession
		}

		if time.Now().After(token.ExpiresAt) {
			return fmt.Errorf("session expired")
		}

		// 2. Generate rotated token early so we can put its hash in the claims.
		newToken, err := generateRandomToken(32)
		if err != nil {
			return fmt.Errorf("failed to generate rotated token: %w", err)
		}
		newHashed := hashToken(newToken)

		// 3. Load user roles/memberships.
		memberships, err := s.repo.GetClubMembershipsByUserUUID(ctx, tx, token.UserUUID)
		if err != nil {
			return fmt.Errorf("failed to load memberships: %w", err)
		}

		var activeUUID uuid.UUID
		var activeRole authdomain.Role = authdomain.RolePlayer
		var clubs []authdomain.ClubRole
		var syncRequests []SyncRequest

		if len(memberships) == 0 {
			memberships = s.backfillClubMemberships(ctx, tx, token.UserUUID)
		}

		if len(memberships) > 0 {
			// Default to first membership.
			activeUUID = memberships[0].ClubUUID
			activeRole = authdomain.Role(memberships[0].Role)

			// If a specific active club was requested, try to find it.
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

			// Get global user for fallback display name.
			var globalDisplayName string
			user, err := s.repo.GetUserByUUID(ctx, tx, token.UserUUID)
			if err == nil && user != nil {
				globalDisplayName = user.GetDisplayName()
			}

			for _, m := range memberships {
				// Request async profile sync if:
				// 1. display_name is NULL (never synced), OR
				// 2. synced_at is NULL or older than ProfileSyncStaleness (stale data).
				needsSync := m.DisplayName == nil ||
					m.SyncedAt == nil ||
					time.Since(*m.SyncedAt) > ProfileSyncStaleness

				// Also trigger sync for role checks on a shorter interval.
				// SyncMember handles both profile and role in a single call.
				needsRoleSync := m.SyncedAt == nil ||
					time.Since(*m.SyncedAt) > RoleSyncStaleness

				if (needsSync || needsRoleSync) && m.ExternalID != nil {
					if gid, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, tx, m.ClubUUID); err == nil {
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

		// Resolve Discord Guild ID from active club.
		// This is critical for NATS permissions if the frontend subscribes using Guild ID.
		var guildID string
		if activeUUID != uuid.Nil {
			if gid, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, tx, activeUUID); err == nil {
				guildID = string(gid)
			} else {
				s.logger.WarnContext(ctx, "Failed to resolve Discord Guild ID for club",
					attr.String("club_uuid", activeUUID.String()),
					attr.Error(err),
				)
				// Fallback to club UUID string if resolution fails, though this may not match frontend expectations.
				guildID = activeUUID.String()
			}
		}

		// 4. Fetch linked providers for this user.
		linkedProviders, err := s.repo.GetLinkedProvidersByUserUUID(ctx, tx, token.UserUUID)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to fetch linked providers",
				attr.String("user_uuid", token.UserUUID.String()),
				attr.Error(err),
			)
			linkedProviders = nil
		}

		// 5. Mint NATS ticket with hash of the new rotated refresh token.
		claims := &authdomain.Claims{
			UserID:           token.UserUUID.String(),
			UserUUID:         token.UserUUID,
			GuildID:          guildID,
			ActiveClubUUID:   activeUUID,
			Role:             activeRole,
			Clubs:            clubs,
			LinkedProviders:  linkedProviders,
			RefreshTokenHash: newHashed,
		}
		natsToken, err := s.jwtProvider.GenerateToken(claims, TicketTTL)
		if err != nil {
			return fmt.Errorf("failed to generate ticket: %w", err)
		}

		// 6. Rotate token atomically inside the same transaction.
		newRefreshToken := &userdb.RefreshToken{
			Hash:        newHashed,
			UserUUID:    token.UserUUID,
			TokenFamily: token.TokenFamily,
			ExpiresAt:   time.Now().UTC().Add(RefreshTokenExpiry),
			Revoked:     false,
		}

		if err := s.repo.RevokeRefreshTokenIfActive(ctx, tx, hashedToken); err != nil {
			if errors.Is(err, userdb.ErrNoRowsAffected) {
				return fmt.Errorf("session revoked")
			}
			return fmt.Errorf("failed to revoke old token: %w", err)
		}

		if err := s.repo.SaveRefreshToken(ctx, tx, newRefreshToken); err != nil {
			return fmt.Errorf("failed to save rotated token: %w", err)
		}

		resp = &TicketResponse{
			NATSToken:    natsToken,
			RefreshToken: newToken,
			SyncRequests: syncRequests,
		}
		return nil
	})
	if err != nil {
		// Revoke all user sessions in a separate transaction so it is not rolled back with the failed ticket flow.
		if errors.Is(err, ErrRevokedSession) && replayedUserUUID != uuid.Nil {
			if revokeErr := s.revokeAllUserTokens(ctx, replayedUserUUID); revokeErr != nil {
				s.logger.ErrorContext(ctx, "Failed to revoke all user tokens after replay detection",
					attr.Error(revokeErr),
					attr.String("user_uuid", replayedUserUUID.String()),
				)
			}
		}
		return nil, err
	}

	return resp, nil
}

func (s *service) revokeAllUserTokens(ctx context.Context, userUUID uuid.UUID) error {
	return s.runInTx(ctx, func(ctx context.Context, tx bun.IDB) error {
		return s.repo.RevokeAllUserTokens(ctx, tx, userUUID)
	})
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
func (s *service) backfillClubMemberships(ctx context.Context, db bun.IDB, userUUID uuid.UUID) []*userdb.ClubMembership {
	user, err := s.repo.GetUserByUUID(ctx, db, userUUID)
	if err != nil || user == nil || user.UserID == nil {
		return nil
	}

	legacyMemberships, err := s.repo.GetUserMemberships(ctx, db, *user.UserID)
	if err != nil || len(legacyMemberships) == 0 {
		return nil
	}

	s.logger.InfoContext(ctx, "Backfilling club memberships for user", attr.String("user_uuid", userUUID.String()))
	for _, lm := range legacyMemberships {
		clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, lm.GuildID)
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
		if err := s.repo.UpsertClubMembership(ctx, db, cm); err != nil {
			s.logger.WarnContext(ctx, "Failed to backfill club membership", attr.Error(err))
		}
	}

	// Re-fetch memberships after backfill
	memberships, _ := s.repo.GetClubMembershipsByUserUUID(ctx, db, userUUID)
	return memberships
}

// resolveDisplayName returns the club display name if present, otherwise fallback.
func resolveDisplayName(clubDisplayName *string, fallback string) string {
	if clubDisplayName != nil && *clubDisplayName != "" {
		return *clubDisplayName
	}
	return fallback
}
