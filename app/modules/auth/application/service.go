package authservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	authjwt "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/jwt"
	authnats "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/nats"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for the auth service.
type Config struct {
	PWABaseURL string
	DefaultTTL time.Duration
}

// service implements the Service interface.
type service struct {
	repo              userdb.Repository
	jwtProvider       authjwt.Provider
	userJWTBuilder    authnats.UserJWTBuilder
	permissionBuilder *permissions.Builder
	config            Config
	logger            *slog.Logger
	tracer            trace.Tracer
}

// NewService creates a new auth service.
func NewService(
	jwtProvider authjwt.Provider,
	userJWTBuilder authnats.UserJWTBuilder,
	repo userdb.Repository,
	config Config,
	logger *slog.Logger,
	tracer trace.Tracer,
) Service {
	return &service{
		repo:              repo,
		jwtProvider:       jwtProvider,
		userJWTBuilder:    userJWTBuilder,
		permissionBuilder: permissions.NewBuilder(),
		config:            config,
		logger:            logger,
		tracer:            tracer,
	}
}

const (
	DefaultTokenTTL    = 24 * time.Hour
	RefreshTokenExpiry = 30 * 24 * time.Hour
)

// GenerateMagicLink generates a magic link URL for the given user and guild.
func (s *service) GenerateMagicLink(ctx context.Context, userID, guildID string, role authdomain.Role) (*MagicLinkResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.GenerateMagicLink")
	defer span.End()

	s.logger.InfoContext(ctx, "Generating magic link request",
		attr.String("user_id", userID),
		attr.String("guild_id", guildID),
		attr.String("role", string(role)),
	)

	// 1. Validate role
	if !role.IsValid() {
		return &MagicLinkResponse{Success: false, Error: ErrInvalidRole.Error()}, nil
	}

	// 2. Resolve internal UUIDs
	userUUID, clubUUID, err := s.resolveUserAndClub(ctx, userID, guildID)
	if err != nil {
		return &MagicLinkResponse{Success: false, Error: "identity resolution failed"}, nil
	}

	// 3. Verify membership
	if err := s.verifyMembership(ctx, userUUID, clubUUID); err != nil {
		return &MagicLinkResponse{Success: false, Error: "unauthorized: user is not a member of the requested club"}, nil
	}

	// 4. Create and save magic link (stateful)
	token, err := s.createAndSaveMagicLink(ctx, userUUID, guildID, role)
	if err != nil {
		return &MagicLinkResponse{Success: false, Error: "failed to generate magic link"}, nil
	}

	// 5. Build URL
	url := s.buildMagicLinkURL(token)

	s.logger.InfoContext(ctx, "Magic link generated successfully",
		attr.String("user_id", userID),
		attr.String("guild_id", guildID),
	)

	return &MagicLinkResponse{Success: true, URL: url}, nil
}

func (s *service) createAndSaveMagicLink(ctx context.Context, userUUID uuid.UUID, guildID string, role authdomain.Role) (string, error) {
	token, err := generateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	ttl := s.config.DefaultTTL
	if ttl == 0 {
		ttl = DefaultTokenTTL
	}

	magicLink := &userdb.MagicLink{
		Token:     token,
		UserUUID:  userUUID,
		GuildID:   guildID,
		Role:      string(role),
		ExpiresAt: time.Now().Add(ttl),
	}

	if err := s.repo.SaveMagicLink(ctx, nil, magicLink); err != nil {
		s.logger.ErrorContext(ctx, "Failed to save magic link", attr.Error(err))
		return "", err
	}

	return token, nil
}

func (s *service) buildMagicLinkURL(token string) string {
	return fmt.Sprintf("%s?t=%s", s.config.PWABaseURL, token)
}

func (s *service) resolveUserAndClub(ctx context.Context, userID, guildID string) (uuid.UUID, uuid.UUID, error) {
	userUUID, err := s.repo.GetUUIDByDiscordID(ctx, nil, sharedtypes.DiscordID(userID))
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to resolve User UUID: %w", err)
	}

	clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, nil, sharedtypes.GuildID(guildID))
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to resolve Club UUID: %w", err)
	}

	return userUUID, clubUUID, nil
}

func (s *service) verifyMembership(ctx context.Context, userUUID, clubUUID uuid.UUID) error {
	// Check if user has membership in the specific club
	// We can use GetClubMembership which is a direct lookup
	_, err := s.repo.GetClubMembership(ctx, nil, userUUID, clubUUID)
	if err != nil {
		return fmt.Errorf("user is not a member of the club")
	}
	return nil
}

// ValidateToken validates a JWT token and returns the claims if valid.
func (s *service) ValidateToken(ctx context.Context, tokenString string) (*authdomain.Claims, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ValidateToken")
	defer span.End()

	if tokenString == "" {
		return nil, ErrMissingToken
	}

	claims, err := s.jwtProvider.ValidateToken(tokenString)
	if err != nil {
		s.logger.WarnContext(ctx, "Token validation failed",
			attr.Error(err),
		)
		return nil, err
	}

	s.logger.DebugContext(ctx, "Token validated successfully",
		attr.String("user_id", claims.UserID),
		attr.String("guild_id", claims.GuildID),
	)

	return claims, nil
}

// HandleNATSAuthRequest processes a NATS auth callout request.
func (s *service) HandleNATSAuthRequest(ctx context.Context, req *NATSAuthRequest) (*NATSAuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.HandleNATSAuthRequest")
	defer span.End()

	s.logger.DebugContext(ctx, "Processing auth callout request",
		attr.String("client_host", req.ClientInfo.Host),
		attr.Any("client_id", req.ClientInfo.ID),
	)

	// Extract and validate the JWT from the password field
	tokenString := req.ConnectOpts.Password
	if tokenString == "" {
		s.logger.WarnContext(ctx, "Auth request missing password/token")
		return &NATSAuthResponse{
			Error: ErrMissingToken.Error(),
		}, nil
	}

	claims, err := s.jwtProvider.ValidateToken(tokenString)
	if err != nil {
		s.logger.WarnContext(ctx, "Token validation failed",
			attr.Error(err),
			attr.String("client_host", req.ClientInfo.Host),
		)
		return &NATSAuthResponse{
			Error: fmt.Sprintf("invalid token: %v", err),
		}, nil
	}

	// Stateful Session Validation:
	// If the ticket was minted from a refresh token (standard PWA flow),
	// check if that refresh token is still valid and not revoked.
	if claims.RefreshTokenHash != "" {
		token, err := s.repo.GetRefreshToken(ctx, nil, claims.RefreshTokenHash)
		if err != nil {
			s.logger.WarnContext(ctx, "Session validation failed: token not found",
				attr.String("rt_hash", claims.RefreshTokenHash),
				attr.String("user_uuid", claims.UserUUID.String()),
			)
			return &NATSAuthResponse{
				Error: ErrSessionMismatch.Error(),
			}, nil
		}

		if token.Revoked {
			s.logger.WarnContext(ctx, "Session validation failed: token revoked",
				attr.String("rt_hash", claims.RefreshTokenHash),
				attr.String("user_uuid", claims.UserUUID.String()),
			)
			return &NATSAuthResponse{
				Error: ErrRevokedSession.Error(),
			}, nil
		}

		if time.Now().After(token.ExpiresAt) {
			s.logger.WarnContext(ctx, "Session validation failed: token expired",
				attr.String("rt_hash", claims.RefreshTokenHash),
				attr.String("user_uuid", claims.UserUUID.String()),
			)
			return &NATSAuthResponse{
				Error: ErrExpiredToken.Error(),
			}, nil
		}
	}

	s.logger.InfoContext(ctx, "Token validated successfully",
		attr.String("user_id", claims.UserID),
		attr.String("guild_id", claims.GuildID),
		attr.String("role", claims.Role.String()),
	)

	// Build permissions based on claims
	perms := s.permissionBuilder.ForRole(claims)

	// Generate NATS user JWT with permissions
	if s.userJWTBuilder == nil {
		s.logger.ErrorContext(ctx, "NATS JWT builder not configured")
		return &NATSAuthResponse{
			Error: ErrGenerateUserJWT.Error(),
		}, nil
	}

	userJWT, err := s.userJWTBuilder.BuildUserJWT(claims, perms)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to generate user JWT",
			attr.Error(err),
			attr.String("user_id", claims.UserID),
		)
		return &NATSAuthResponse{
			Error: ErrGenerateUserJWT.Error(),
		}, nil
	}

	return &NATSAuthResponse{
		Jwt: userJWT,
	}, nil
}
