package authservice

import (
	"context"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
)

// Service defines the authentication service interface.
type Service interface {
	// GenerateMagicLink generates a magic link URL for the given user and guild.
	GenerateMagicLink(ctx context.Context, userID, guildID string, role authdomain.Role) (*MagicLinkResponse, error)

	// ValidateToken validates a JWT token and returns the claims if valid.
	ValidateToken(ctx context.Context, tokenString string) (*authdomain.Claims, error)

	// HandleNATSAuthRequest processes a NATS auth callout request.
	HandleNATSAuthRequest(ctx context.Context, req *NATSAuthRequest) (*NATSAuthResponse, error)

	// LoginUser validates a one-time token and creates a long-lived session (refresh token).
	LoginUser(ctx context.Context, oneTimeToken string) (*LoginResponse, error)

	// GetTicket validates a refresh token and mints a short-lived NATS ticket.
	GetTicket(ctx context.Context, refreshToken string) (*TicketResponse, error)

	// LogoutUser revokes a refresh token.
	LogoutUser(ctx context.Context, refreshToken string) error
}

type LoginResponse struct {
	RefreshToken string
	UserUUID     string
}

type TicketResponse struct {
	NATSToken    string
	RefreshToken string
}

// NATSAuthRequest represents a NATS auth callout request.
type NATSAuthRequest struct {
	UserNkey        string         `json:"user_nkey"`
	ServerPublicKey string         `json:"server_public_key"` // The server's public key (aud for response)
	ConnectOpts     ConnectOptions `json:"connect_opts"`
	ClientInfo      ClientInfo     `json:"client_info"`
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

// NATSAuthResponse represents the response to a NATS auth callout.
type NATSAuthResponse struct {
	Jwt            string `json:"jwt,omitempty"`             // The user JWT (for logging)
	Error          string `json:"error,omitempty"`           // Error message if auth failed
	SignedResponse string `json:"signed_response,omitempty"` // The signed auth response JWT to send to NATS
}

// MagicLinkResponse represents the response for magic link generation
type MagicLinkResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}
