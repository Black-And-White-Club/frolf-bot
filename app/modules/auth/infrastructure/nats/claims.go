package authnats

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nkeys"
)

// UserClaims represents NATS user JWT claims.
type UserClaims struct {
	Subject          string          `json:"sub"`
	Audience         string          `json:"aud,omitempty"`
	Expires          int64           `json:"exp,omitempty"`
	IssuedAt         int64           `json:"iat"`
	Issuer           string          `json:"iss"`
	Name             string          `json:"name,omitempty"`
	RefreshTokenHash string          `json:"rt_hash,omitempty"`
	Nats             UserPermissions `json:"nats"`
}

// UserPermissions contains the NATS permissions for a user.
// According to NATS JWT spec, type, version, and issuer_account go inside the nats object.
type UserPermissions struct {
	Pub           PermissionRules `json:"pub,omitempty"`
	Sub           PermissionRules `json:"sub,omitempty"`
	Resp          *RespPermission `json:"resp,omitempty"`
	IssuerAccount string          `json:"issuer_account,omitempty"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
}

// PermissionRules contains allow/deny patterns.
type PermissionRules struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// RespPermission allows request/reply patterns.
type RespPermission struct {
	Max int `json:"max,omitempty"`
	TTL int `json:"ttl,omitempty"`
}

// AuthorizationResponseClaims represents the claims in an auth callout response JWT.
type AuthorizationResponseClaims struct {
	Audience string                       `json:"aud,omitempty"`
	IssuedAt int64                        `json:"iat"`
	Issuer   string                       `json:"iss"`
	Subject  string                       `json:"sub"`
	Nats     AuthorizationResponsePayload `json:"nats"`
}

// AuthorizationResponsePayload contains the NATS-specific response data.
type AuthorizationResponsePayload struct {
	JWT     string `json:"jwt,omitempty"`
	Error   string `json:"error,omitempty"`
	Account string `json:"account,omitempty"`
	Type    string `json:"type"`
	Version int    `json:"version"`
}

// NewAuthorizationResponseClaims creates new authorization response claims.
func NewAuthorizationResponseClaims(audience string, subject string, issuerAccount string, userJWT string, errMsg string) *AuthorizationResponseClaims {
	return &AuthorizationResponseClaims{
		Audience: audience,
		IssuedAt: time.Now().Unix(),
		Subject:  subject, // Subject must be the user's public key
		Nats: AuthorizationResponsePayload{
			JWT:     userJWT,
			Error:   errMsg,
			Account: issuerAccount,
			Type:    "authorization_response",
			Version: 2,
		},
	}
}

// Encode encodes the authorization response claims as a signed JWT.
func (c *AuthorizationResponseClaims) Encode(kp nkeys.KeyPair) (string, error) {
	// Get issuer public key
	issuer, err := kp.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get issuer public key: %w", err)
	}
	c.Issuer = issuer

	// Create header
	header := map[string]string{
		"typ": "JWT",
		"alg": "ed25519-nkey",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64 URL encode
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signing input
	signingInput := headerB64 + "." + claimsB64

	// Sign
	sig, err := kp.Sign([]byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}

// NewUserClaims creates a new UserClaims with defaults.
func NewUserClaims(publicKey string) *UserClaims {
	return &UserClaims{
		Subject:  publicKey,
		IssuedAt: time.Now().Unix(),
		Nats: UserPermissions{
			Type:    "user",
			Version: 2,
		},
	}
}

// Encode encodes the claims as a signed JWT.
func (c *UserClaims) Encode(kp nkeys.KeyPair) (string, error) {
	// Get issuer public key
	issuer, err := kp.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get issuer public key: %w", err)
	}
	c.Issuer = issuer

	// Create header
	header := map[string]string{
		"typ": "JWT",
		"alg": "ed25519-nkey",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64 URL encode
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signing input
	signingInput := headerB64 + "." + claimsB64

	// Sign
	sig, err := kp.Sign([]byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}
