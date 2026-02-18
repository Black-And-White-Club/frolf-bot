package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	authoauth "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/oauth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// Config holds the Google OAuth2 application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Provider implements the oauth.Provider interface for Google.
type Provider struct {
	cfg oauth2.Config
}

// NewProvider creates a Google OAuth2 provider from the given config.
func NewProvider(cfg Config) *Provider {
	return &Provider{
		cfg: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"openid", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// Name returns the stable provider name.
func (p *Provider) Name() string { return "google" }

// AuthCodeURL returns the Google authorization URL.
func (p *Provider) AuthCodeURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange exchanges an authorization code for user identity.
func (p *Provider) Exchange(ctx context.Context, code string) (*authoauth.UserInfo, error) {
	token, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google: token exchange failed: %w", err)
	}

	client := p.cfg.Client(ctx, token)
	userInfo, err := fetchGoogleUser(client)
	if err != nil {
		return nil, err
	}

	info := &authoauth.UserInfo{
		Provider:    "google",
		ProviderID:  userInfo.Sub,
		DisplayName: resolveDisplayName(userInfo),
		AccessToken: token.AccessToken,
	}
	if !token.Expiry.IsZero() {
		expiry := token.Expiry
		info.AccessTokenExpiresAt = &expiry
	}
	return info, nil
}

// --- internal helpers ---

type googleUserResponse struct {
	Sub        string `json:"sub"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email      string `json:"email"`
	Picture    string `json:"picture"`
}

func fetchGoogleUser(client *http.Client) (*googleUserResponse, error) {
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("google: failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google: unexpected status %d fetching user", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google: failed to read user response: %w", err)
	}

	var user googleUserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("google: failed to parse user response: %w", err)
	}

	return &user, nil
}

func resolveDisplayName(u *googleUserResponse) string {
	if u.Name != "" {
		return u.Name
	}
	return u.GivenName
}
