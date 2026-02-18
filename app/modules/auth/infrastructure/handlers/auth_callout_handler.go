package authhandlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

var (
	// ErrInvalidJWTFormat indicates the JWT doesn't have the expected format.
	ErrInvalidJWTFormat = errors.New("invalid JWT format: expected 3 parts")
	// ErrInvalidJWTPayload indicates the JWT payload couldn't be decoded.
	ErrInvalidJWTPayload = errors.New("invalid JWT payload: base64 decode failed")
)

// AuthorizationRequestClaims represents the claims in a NATS auth callout JWT.
type AuthorizationRequestClaims struct {
	Audience string      `json:"aud,omitempty"`
	Expires  int64       `json:"exp,omitempty"`
	IssuedAt int64       `json:"iat,omitempty"`
	Issuer   string      `json:"iss,omitempty"`
	Subject  string      `json:"sub,omitempty"`
	Nats     NatsRequest `json:"nats,omitempty"`
}

// NatsRequest contains the NATS-specific data in the auth callout request.
type NatsRequest struct {
	UserNkey    string                 `json:"user_nkey,omitempty"`
	ConnectOpts ConnectOptsFromJWT     `json:"connect_opts,omitempty"`
	ClientInfo  authservice.ClientInfo `json:"client_info,omitempty"`
}

// ConnectOptsFromJWT represents connect options in the auth request JWT.
type ConnectOptsFromJWT struct {
	JWT      string `json:"jwt,omitempty"`
	Token    string `json:"auth_token,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"pass,omitempty"`
	Name     string `json:"name,omitempty"`
	Lang     string `json:"lang,omitempty"`
	Version  string `json:"version,omitempty"`
	Protocol int    `json:"protocol,omitempty"`
}

// HandleNATSAuthCallout processes an auth callout message from NATS.
func (h *AuthHandlers) HandleNATSAuthCallout(msg *nats.Msg) {
	ctx, span := h.tracer.Start(context.Background(), "AuthHandlers.HandleNATSAuthCallout")
	defer span.End()

	h.logger.DebugContext(ctx, "Received auth callout request",
		attr.String("subject", msg.Subject),
		attr.Int("data_length", len(msg.Data)),
	)
	// The auth callout message is a JWT - decode it
	claims, err := h.decodeAuthRequestJWT(string(msg.Data))
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to decode auth request JWT",
			attr.Error(err),
		)
		h.respondWithError(msg, "invalid request format")
		return
	}

	h.logger.DebugContext(ctx, "Decoded auth request",
		attr.String("subject", claims.Subject),
		attr.String("issuer", claims.Issuer), // Log the issuer
		attr.String("client_nkey", claims.Nats.UserNkey),
		attr.String("client_name", claims.Nats.ConnectOpts.Name),
	)

	// Convert to the service request format
	// The user's JWT token can be in either the JWT field or Password field
	userToken := claims.Nats.ConnectOpts.JWT
	if userToken == "" {
		userToken = claims.Nats.ConnectOpts.Password
	}
	if userToken == "" {
		userToken = claims.Nats.ConnectOpts.Token
	}
	if userToken == "" {
		h.logger.DebugContext(ctx, "Auth token not found in any standard field",
			attr.String("client_nkey", claims.Nats.UserNkey),
		)
	}

	req := &authservice.NATSAuthRequest{
		UserNkey:        claims.Nats.UserNkey,
		ServerPublicKey: claims.Issuer, // The server's public key (iss) not the account (sub)
		ConnectOpts: authservice.ConnectOptions{
			Password: userToken,
			User:     claims.Nats.ConnectOpts.User,
		},
		ClientInfo: claims.Nats.ClientInfo,
	}

	// Process the auth request
	resp, err := h.service.HandleNATSAuthRequest(ctx, req)
	if err != nil {
		h.logger.ErrorContext(ctx, "Auth request processing failed",
			attr.Error(err),
		)
		h.respondWithError(msg, "internal error")
		return
	}

	// Send signed response JWT (not plain JSON)
	if resp.SignedResponse == "" {
		h.logger.ErrorContext(ctx, "No signed response generated")
		h.respondWithError(msg, "internal error")
		return
	}

	if err := msg.Respond([]byte(resp.SignedResponse)); err != nil {
		h.logger.ErrorContext(ctx, "Failed to send auth response",
			attr.Error(err),
		)
		return
	}

	if resp.Error != "" {
		h.logger.WarnContext(ctx, "Auth request denied",
			attr.String("error", resp.Error),
		)
	} else {
		h.logger.InfoContext(ctx, "Auth request approved",
			attr.String("user_jwt_fingerprint", tokenFingerprint(resp.Jwt)),
			attr.String("signed_response_fingerprint", tokenFingerprint(resp.SignedResponse)),
		)
	}
}

func tokenFingerprint(token string) string {
	if token == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:6])
}

// decodeAuthRequestJWT decodes a NATS auth callout JWT and extracts the claims.
// When serverPublicKey is configured, the JWT signature is verified using the
// NATS server's public NKey before the claims are trusted.
func (h *AuthHandlers) decodeAuthRequestJWT(token string) (*AuthorizationRequestClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidJWTFormat
	}

	// Verify signature when a server public key is configured.
	if h.serverPublicKey != "" {
		kp, err := nkeys.FromPublicKey(h.serverPublicKey)
		if err != nil {
			return nil, fmt.Errorf("invalid server public key config: %w", err)
		}
		sig, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid jwt signature encoding: %w", err)
		}
		if err := kp.Verify([]byte(parts[0]+"."+parts[1]), sig); err != nil {
			return nil, fmt.Errorf("jwt signature verification failed: %w", err)
		}
	}

	// Decode the payload (claims).
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try standard base64 with padding
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, ErrInvalidJWTPayload
		}
	}

	var claims AuthorizationRequestClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// respondWithError sends an error response.
func (h *AuthHandlers) respondWithError(msg *nats.Msg, errMsg string) {
	resp := authservice.NATSAuthResponse{
		Error: errMsg,
	}
	respData, _ := json.Marshal(resp)
	if err := msg.Respond(respData); err != nil {
		h.logger.Error("Failed to send error response",
			attr.Error(err),
		)
	}
}
