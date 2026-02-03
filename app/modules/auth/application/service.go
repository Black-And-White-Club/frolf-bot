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

	s.logger.InfoContext(ctx, "Generating magic link",
		attr.String("user_id", userID),
		attr.String("guild_id", guildID),
		attr.String("role", role.String()),
	)

	if !role.IsValid() {
		s.logger.WarnContext(ctx, "Invalid role specified",
			attr.String("role", role.String()),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   ErrInvalidRole.Error(),
		}, nil
	}

	// Resolve UUIDs
	userUUID, err := s.repo.GetUUIDByDiscordID(ctx, nil, sharedtypes.DiscordID(userID))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to resolve User UUID",
			attr.Error(err),
			attr.String("user_id", userID),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   "identity resolution failed",
		}, nil
	}

	clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, nil, sharedtypes.GuildID(guildID))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to resolve Club UUID",
			attr.Error(err),
			attr.String("guild_id", guildID),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   "club resolution failed",
		}, nil
	}

	// Fetch all club memberships for the user
	memberships, err := s.repo.GetClubMembershipsByUserUUID(ctx, nil, userUUID)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to fetch club memberships",
			attr.Error(err),
			attr.String("user_uuid", userUUID.String()),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   "failed to fetch club memberships",
		}, nil
	}

	clubs := make([]authdomain.ClubRole, 0, len(memberships))
	foundActive := false
	for _, m := range memberships {
		clubRole := authdomain.ClubRole{
			ClubUUID: m.ClubUUID,
			Role:     authdomain.Role(m.Role),
		}
		clubs = append(clubs, clubRole)
		if m.ClubUUID == clubUUID {
			foundActive = true
		}
	}

	// Based on the review "doesn't verify active club is actually valid for user",
	// let's ensure it's valid.
	if !foundActive {
		s.logger.WarnContext(ctx, "User is not a member of the requested club",
			attr.String("user_uuid", userUUID.String()),
			attr.String("club_uuid", clubUUID.String()),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   "unauthorized: user is not a member of the requested club",
		}, nil
	}

	// Generate the token using the default TTL
	ttl := s.config.DefaultTTL
	if ttl == 0 {
		ttl = DefaultTokenTTL
	}

	claims := &authdomain.Claims{
		UserID:         userID,
		UserUUID:       userUUID,
		ActiveClubUUID: clubUUID,
		GuildID:        guildID,
		Role:           role,
		Clubs:          clubs,
	}

	token, err := s.jwtProvider.GenerateToken(claims, ttl)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to generate token",
			attr.Error(err),
			attr.String("user_id", userID),
		)
		return &MagicLinkResponse{
			Success: false,
			Error:   ErrGenerateToken.Error(),
		}, nil
	}

	// Build the magic link URL
	url := fmt.Sprintf("%s?t=%s", s.config.PWABaseURL, token)

	s.logger.InfoContext(ctx, "Magic link generated successfully",
		attr.String("user_id", userID),
		attr.String("guild_id", guildID),
	)

	return &MagicLinkResponse{
		Success: true,
		URL:     url,
	}, nil
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
