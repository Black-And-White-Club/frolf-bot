package authcallout

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nkeys"
)

// UserClaims represents NATS user JWT claims.
type UserClaims struct {
	Subject     string          `json:"sub"`
	Audience    string          `json:"aud,omitempty"`
	Expires     int64           `json:"exp,omitempty"`
	IssuedAt    int64           `json:"iat"`
	Issuer      string          `json:"iss"`
	Name        string          `json:"name,omitempty"`
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	Permissions UserPermissions `json:"nats"`
}

// UserPermissions contains the NATS permissions for a user.
type UserPermissions struct {
	Pub  PermissionRules `json:"pub,omitempty"`
	Sub  PermissionRules `json:"sub,omitempty"`
	Resp *RespPermission `json:"resp,omitempty"`
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

// NewUserClaims creates a new UserClaims with defaults.
func NewUserClaims(publicKey string) *UserClaims {
	return &UserClaims{
		Subject:  publicKey,
		IssuedAt: time.Now().Unix(),
		Type:     "user",
		Version:  2,
		Permissions: UserPermissions{
			Resp: &RespPermission{
				Max: 1,
				TTL: 5000,
			},
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
