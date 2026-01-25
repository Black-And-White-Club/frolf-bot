package authcallout

import "context"

// Service defines the auth callout service interface.
type Service interface {
	HandleAuthRequest(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
}

// AuthRequest represents a NATS auth callout request.
type AuthRequest struct {
	ConnectOpts ConnectOptions `json:"connect_opts"`
	ClientInfo  ClientInfo     `json:"client_info"`
}

// ConnectOptions contains the connection options from the auth request.
type ConnectOptions struct {
	Password string `json:"pass"` // Contains the JWT
	User     string `json:"user,omitempty"`
}

// ClientInfo contains client information from the auth request.
type ClientInfo struct {
	Host string `json:"host,omitempty"`
	ID   uint64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// AuthResponse represents the response to a NATS auth callout.
type AuthResponse struct {
	Jwt   string `json:"jwt,omitempty"`
	Error string `json:"error,omitempty"`
}

// Permissions defines pub/sub permissions for a user.
type Permissions struct {
	Subscribe PermissionSet `json:"subscribe"`
	Publish   PermissionSet `json:"publish"`
}

// PermissionSet contains allow and deny patterns.
type PermissionSet struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny,omitempty"`
}
