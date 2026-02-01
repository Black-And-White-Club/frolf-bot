package authservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	authjwt "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/jwt"
	authnats "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/nats"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for the auth service.
type Config struct {
	PWABaseURL string
	DefaultTTL time.Duration
}

// service implements the Service interface.
type service struct {
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
	config Config,
	logger *slog.Logger,
	tracer trace.Tracer,
) Service {
	return &service{
		jwtProvider:       jwtProvider,
		userJWTBuilder:    userJWTBuilder,
		permissionBuilder: permissions.NewBuilder(),
		config:            config,
		logger:            logger,
		tracer:            tracer,
	}
}

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

	// Generate the token using the default TTL
	ttl := s.config.DefaultTTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	token, err := s.jwtProvider.GenerateToken(userID, guildID, role, ttl)
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

	// Build permissions based on role
	perms := s.permissionBuilder.ForRole(claims.Role, claims.GuildID, claims.UserID)

	// Generate NATS user JWT with permissions
	if s.userJWTBuilder == nil {
		s.logger.ErrorContext(ctx, "NATS JWT builder not configured")
		return &NATSAuthResponse{
			Error: ErrGenerateUserJWT.Error(),
		}, nil
	}

	userJWT, err := s.userJWTBuilder.BuildUserJWT(claims.UserID, claims.GuildID, perms)
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
