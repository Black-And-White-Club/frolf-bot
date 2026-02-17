package roundservice

import (
	"fmt"
	"net/url"
	"strings"
)

var allowedUDiscHosts = map[string]struct{}{
	"udisc.com":     {},
	"www.udisc.com": {},
}

func validateUDiscURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("invalid URL")
	}

	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("unsupported URL scheme")
	}

	if u.User != nil {
		return fmt.Errorf("userinfo is not allowed")
	}

	if u.Port() != "" {
		return fmt.Errorf("explicit ports are not allowed")
	}

	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if _, ok := allowedUDiscHosts[host]; !ok {
		return fmt.Errorf("unsupported host: %s", u.Host)
	}

	return nil
}

func parseAndValidateUDiscURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}

	if err := validateUDiscURL(u); err != nil {
		return nil, err
	}

	return u, nil
}

// normalizeUDiscExportURL canonicalizes a variety of UDisc leaderboard URLs to the
// canonical /leaderboard/export path and performs a strict host allowlist check.
func normalizeUDiscExportURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("empty URL")
	}

	u, err := parseAndValidateUDiscURL(raw)
	if err != nil {
		return "", err
	}

	// Strip browser-only parts
	u.RawQuery = ""
	u.Fragment = ""

	path := strings.TrimSuffix(u.Path, "/")

	switch {
	case strings.HasSuffix(path, "/leaderboard/export"):
		// already canonical
	case strings.HasSuffix(path, "/leaderboard"):
		path += "/export"
	case strings.Contains(path, "/leaderboard/"):
		path = strings.Split(path, "/leaderboard/")[0] + "/leaderboard/export"
	default:
		return "", fmt.Errorf("not a UDisc leaderboard URL")
	}

	u.Path = path
	return u.String(), nil
}
