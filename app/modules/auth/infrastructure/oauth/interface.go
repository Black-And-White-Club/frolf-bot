package oauth

import (
	"context"
	"time"
)

// Provider defines the interface for an OAuth2 identity provider.
type Provider interface {
	// Name returns the provider's stable identifier (e.g. "discord", "google").
	Name() string

	// AuthCodeURL returns the redirect URL to begin the OAuth2 flow.
	AuthCodeURL(state string) string

	// Exchange exchanges an authorization code for provider-specific user info.
	Exchange(ctx context.Context, code string) (*UserInfo, error)
}

// UserInfo holds the normalized identity returned by any provider after a successful exchange.
type UserInfo struct {
	Provider             string     // stable provider name, e.g. "discord"
	ProviderID           string     // provider's stable user identifier (Discord snowflake, Google sub, etc.)
	DisplayName          string     // cached display name at time of linking
	AccessToken          string     // OAuth2 access token (optional; stored for API calls)
	AccessTokenExpiresAt *time.Time // when the access token expires (nil if unknown)
}
