package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	authoauth "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/oauth"
	"golang.org/x/oauth2"
)

const (
	discordAuthURL  = "https://discord.com/api/oauth2/authorize"
	discordTokenURL = "https://discord.com/api/oauth2/token"
	discordAPIBase  = "https://discord.com/api/v10"
)

// Config holds the Discord OAuth2 application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GuildInfo represents a Discord server returned by the /users/@me/guilds endpoint.
type GuildInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"` // string-encoded int64 in Discord's API
}

// HasManageGuild reports whether the user has MANAGE_GUILD (0x20) or ADMINISTRATOR (0x8) permission.
func (g *GuildInfo) HasManageGuild() bool {
	perms, err := strconv.ParseInt(g.Permissions, 10, 64)
	if err != nil {
		return false
	}
	return (perms&0x20) != 0 || (perms&0x8) != 0
}

// Provider implements the oauth.Provider interface for Discord.
// It holds two oauth2.Config values — one for login (identify scope) and one for
// guild linking (identify + guilds scope).
type Provider struct {
	loginCfg oauth2.Config
	guildCfg oauth2.Config
}

// NewProvider creates a Discord OAuth2 provider from the given config.
func NewProvider(cfg Config) *Provider {
	base := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  discordAuthURL,
			TokenURL: discordTokenURL,
		},
	}

	login := base
	login.Scopes = []string{"identify", "guilds"}

	guild := base
	guild.Scopes = []string{"identify", "guilds"}

	return &Provider{
		loginCfg: login,
		guildCfg: guild,
	}
}

// Name returns the stable provider name.
func (p *Provider) Name() string { return "discord" }

// AuthCodeURL returns the Discord authorization URL for the login (identify) flow.
func (p *Provider) AuthCodeURL(state string) string {
	return p.loginCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// GuildAuthCodeURL returns the Discord authorization URL for the guild-linking flow
// (identify + guilds scopes).
func (p *Provider) GuildAuthCodeURL(state string) string {
	return p.guildCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange exchanges an authorization code for user identity via the identify scope.
func (p *Provider) Exchange(ctx context.Context, code string) (*authoauth.UserInfo, error) {
	token, err := p.loginCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("discord: token exchange failed: %w", err)
	}

	client := p.loginCfg.Client(ctx, token)
	discordUser, err := fetchDiscordUser(client)
	if err != nil {
		return nil, err
	}

	info := &authoauth.UserInfo{
		Provider:    "discord",
		ProviderID:  discordUser.ID,
		DisplayName: resolveDisplayName(discordUser),
		AccessToken: token.AccessToken,
	}
	if !token.Expiry.IsZero() {
		expiry := token.Expiry
		info.AccessTokenExpiresAt = &expiry
	}
	return info, nil
}

// ExchangeWithGuilds exchanges an authorization code using the guilds scope and
// returns the user identity plus the list of guilds where the user has manage permissions.
func (p *Provider) ExchangeWithGuilds(ctx context.Context, code string) (*authoauth.UserInfo, []GuildInfo, error) {
	token, err := p.guildCfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("discord: guild token exchange failed: %w", err)
	}

	client := p.guildCfg.Client(ctx, token)

	discordUser, err := fetchDiscordUser(client)
	if err != nil {
		return nil, nil, err
	}

	userInfo := &authoauth.UserInfo{
		Provider:    "discord",
		ProviderID:  discordUser.ID,
		DisplayName: resolveDisplayName(discordUser),
	}

	guilds, err := fetchManageableGuilds(client)
	if err != nil {
		return nil, nil, err
	}

	return userInfo, guilds, nil
}

// --- internal helpers ---

type discordUserResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
}

func fetchDiscordUser(client *http.Client) (*discordUserResponse, error) {
	resp, err := client.Get(discordAPIBase + "/users/@me")
	if err != nil {
		return nil, fmt.Errorf("discord: failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord: unexpected status %d fetching user", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("discord: failed to read user response: %w", err)
	}

	var user discordUserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("discord: failed to parse user response: %w", err)
	}

	return &user, nil
}

func fetchManageableGuilds(client *http.Client) ([]GuildInfo, error) {
	resp, err := client.Get(discordAPIBase + "/users/@me/guilds")
	if err != nil {
		return nil, fmt.Errorf("discord: failed to fetch guilds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord: unexpected status %d fetching guilds", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("discord: failed to read guilds response: %w", err)
	}

	var all []GuildInfo
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, fmt.Errorf("discord: failed to parse guilds response: %w", err)
	}

	var manageable []GuildInfo
	for _, g := range all {
		if g.HasManageGuild() {
			manageable = append(manageable, g)
		}
	}

	return manageable, nil
}

func resolveDisplayName(u *discordUserResponse) string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}
