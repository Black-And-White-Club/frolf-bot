package authcallout

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/authcallout/infrastructure/permissions"
	"github.com/Black-And-White-Club/frolf-bot/pkg/jwt"
	"github.com/nats-io/nkeys"
	"go.opentelemetry.io/otel/trace"
)

// AuthCalloutService implements the Service interface.
type AuthCalloutService struct {
	jwtService        jwt.Service
	permissionBuilder *permissions.Builder
	signingKey        nkeys.KeyPair
	issuerAccount     string
	logger            *slog.Logger
	tracer            trace.Tracer
}

// NewService creates a new AuthCalloutService.
func NewService(
	jwtService jwt.Service,
	signingKey nkeys.KeyPair,
	issuerAccount string,
	logger *slog.Logger,
	tracer trace.Tracer,
) *AuthCalloutService {
	return &AuthCalloutService{
		jwtService:        jwtService,
		permissionBuilder: permissions.NewBuilder(),
		signingKey:        signingKey,
		issuerAccount:     issuerAccount,
		logger:            logger,
		tracer:            tracer,
	}
}

// HandleAuthRequest processes a NATS auth callout request.
func (s *AuthCalloutService) HandleAuthRequest(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthCalloutService.HandleAuthRequest")
	defer span.End()

	s.logger.DebugContext(ctx, "Processing auth callout request",
		attr.String("client_host", req.ClientInfo.Host),
		attr.Any("client_id", req.ClientInfo.ID),
	)

	// Extract and validate the JWT from the password field
	tokenString := req.ConnectOpts.Password
	if tokenString == "" {
		s.logger.WarnContext(ctx, "Auth request missing password/token")
		return &AuthResponse{
			Error: "missing authentication token",
		}, nil
	}

	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		s.logger.WarnContext(ctx, "Token validation failed",
			attr.Error(err),
			attr.String("client_host", req.ClientInfo.Host),
		)
		return &AuthResponse{
			Error: fmt.Sprintf("invalid token: %v", err),
		}, nil
	}

	s.logger.InfoContext(ctx, "Token validated successfully",
		attr.String("user_id", claims.Subject),
		attr.String("guild_id", claims.Guild),
		attr.String("role", claims.Role),
	)

	// Build permissions based on role
	role := jwt.Role(claims.Role)
	perms := s.permissionBuilder.ForRole(role, claims.Guild, claims.Subject)

	// Generate NATS user JWT with permissions
	userJWT, err := s.generateUserJWT(claims.Subject, claims.Guild, perms)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to generate user JWT",
			attr.Error(err),
			attr.String("user_id", claims.Subject),
		)
		return &AuthResponse{
			Error: "internal error generating credentials",
		}, nil
	}

	return &AuthResponse{
		Jwt: userJWT,
	}, nil
}

// generateUserJWT creates a NATS user JWT with the specified permissions.
func (s *AuthCalloutService) generateUserJWT(userID, guildID string, perms *permissions.Permissions) (string, error) {
	// Get public key from signing key
	publicKey, err := s.signingKey.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Create user claims for NATS
	uc := NewUserClaims(publicKey)
	uc.Name = fmt.Sprintf("%s@%s", userID, guildID)
	uc.Audience = s.issuerAccount
	uc.Expires = time.Now().Add(24 * time.Hour).Unix()

	// Set permissions
	uc.Permissions.Pub.Allow = perms.Publish.Allow
	uc.Permissions.Pub.Deny = perms.Publish.Deny
	uc.Permissions.Sub.Allow = perms.Subscribe.Allow
	uc.Permissions.Sub.Deny = perms.Subscribe.Deny

	// Encode and sign the JWT
	token, err := uc.Encode(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode user claims: %w", err)
	}

	return token, nil
}
