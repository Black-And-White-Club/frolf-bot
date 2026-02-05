package authhandlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/nats-io/nats.go"
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
			attr.Any("connect_opts", claims.Nats.ConnectOpts),
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
			attr.String("jwt", resp.Jwt),
			attr.String("signed_response", resp.SignedResponse),
		)
	}
}

// decodeAuthRequestJWT decodes a NATS auth callout JWT and extracts the claims.
// Signature verification is intentionally skipped because NATS server is a trusted
// sender and has already validated the request before forwarding it via the auth
// callout mechanism.
func (h *AuthHandlers) decodeAuthRequestJWT(token string) (*AuthorizationRequestClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidJWTFormat
	}

	// Decode the payload (claims) — signature not verified; see function comment.
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
