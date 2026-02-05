package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
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

	if len(memberships) == 0 {
		// Backfill Check: If user has no club memberships, check for legacy guild memberships
		// and create corresponding club memberships if possible.
		user, err := s.repo.GetUserByUUID(ctx, nil, token.UserUUID)
		if err == nil && user != nil && user.UserID != nil {
			legacyMemberships, err := s.repo.GetUserMemberships(ctx, nil, *user.UserID)
			if err == nil && len(legacyMemberships) > 0 {
				s.logger.InfoContext(ctx, "Backfilling club memberships for user", attr.String("user_uuid", token.UserUUID.String()))
				for _, lm := range legacyMemberships {
					clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, nil, lm.GuildID)
					if err == nil {
						extID := string(lm.UserID)
						emptyName := ""
						cm := &userdb.ClubMembership{
							ClubUUID:    clubUUID,
							UserUUID:    token.UserUUID,
							Role:        lm.Role,
							JoinedAt:    lm.JoinedAt,
							ExternalID:  &extID,
							DisplayName: &emptyName,
						}
						if err := s.repo.UpsertClubMembership(ctx, nil, cm); err != nil {
							s.logger.WarnContext(ctx, "Failed to backfill club membership", attr.Error(err))
						}
					}
				}
				// Re-fetch memberships after backfill
				memberships, _ = s.repo.GetClubMembershipsByUserUUID(ctx, nil, token.UserUUID)
			}
		}
	}

	if len(memberships) > 0 {
		activeClubUUID = memberships[0].ClubUUID
		activeRole = authdomain.Role(memberships[0].Role)

		// Get global user for fallback display name
		var globalDisplayName string
		user, err := s.repo.GetUserByUUID(ctx, nil, token.UserUUID)
		if err == nil && user != nil && user.DisplayName != nil {
			globalDisplayName = *user.DisplayName
		}

		for _, m := range memberships {
			// Request async profile sync if:
			// 1. display_name is NULL (never synced), OR
			// 2. synced_at is NULL or older than ProfileSyncStaleness (stale data)
			needsSync := m.DisplayName == nil ||
				m.SyncedAt == nil ||
				time.Since(*m.SyncedAt) > ProfileSyncStaleness

			if needsSync && m.ExternalID != nil {
				s.requestProfileSync(ctx, *m.ExternalID, m.ClubUUID)
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
	if activeClubUUID != uuid.Nil {
		if gid, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, nil, activeClubUUID); err == nil {
			guildID = string(gid)
		} else {
			s.logger.WarnContext(ctx, "Failed to resolve Discord Guild ID for club",
				attr.String("club_uuid", activeClubUUID.String()),
				attr.Error(err),
			)
			// Fallback to club UUID string if resolution fails, though this may not match frontend expectations
			guildID = activeClubUUID.String()
		}
	}

	// 4. Mint NATS Ticket with the hash of the NEW rotated refresh token
	claims := &authdomain.Claims{
		UserID:           token.UserUUID.String(), // Use UUID as ID for now since we are decoupling
		UserUUID:         token.UserUUID,
		GuildID:          guildID,
		ActiveClubUUID:   activeClubUUID,
		Role:             activeRole,
		Clubs:            clubs,
		RefreshTokenHash: newHashed,
	}

	// 4. Mint standard HMAC JWT as a "ticket"
	// This ticket will be exchanged for a NATS User JWT via Auth Callout
	natsToken, err := s.jwtProvider.GenerateToken(claims, DefaultTokenTTL)
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

// resolveDisplayName returns the club display name if present, otherwise fallback.
func resolveDisplayName(clubDisplayName *string, fallback string) string {
	if clubDisplayName != nil && *clubDisplayName != "" {
		return *clubDisplayName
	}
	return fallback
}

// requestProfileSync publishes an async request to sync a user's profile from Discord.
// This is fire-and-forget - failures are logged but don't block the ticket response.
func (s *service) requestProfileSync(ctx context.Context, discordUserID string, clubUUID uuid.UUID) {
	if s.eventBus == nil {
		return
	}

	// Resolve guild ID from club UUID
	guildID, err := s.repo.GetDiscordGuildIDByClubUUID(ctx, nil, clubUUID)
	if err != nil {
		s.logger.DebugContext(ctx, "Cannot request profile sync: failed to resolve guild ID",
			attr.String("club_uuid", clubUUID.String()),
			attr.Error(err),
		)
		return
	}

	payload := &userevents.UserProfileSyncRequestPayloadV1{
		UserID:  sharedtypes.DiscordID(discordUserID),
		GuildID: guildID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to marshal profile sync request",
			attr.Error(err),
		)
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("topic", userevents.UserProfileSyncRequestTopicV1)
	msg.Metadata.Set("user_id", discordUserID)
	msg.Metadata.Set("guild_id", string(guildID))

	if err := s.eventBus.Publish(userevents.UserProfileSyncRequestTopicV1, msg); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish profile sync request",
			attr.Error(err),
			attr.String("user_id", discordUserID),
			attr.String("guild_id", string(guildID)),
		)
		return
	}

	s.logger.InfoContext(ctx, "Published profile sync request",
		attr.String("user_id", discordUserID),
		attr.String("guild_id", string(guildID)),
	)
}
