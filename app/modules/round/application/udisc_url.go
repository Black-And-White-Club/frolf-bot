package roundservice

import (
	"fmt"
	"net/url"
	"strings"
)

// normalizeUDiscExportURL canonicalizes a variety of UDisc leaderboard URLs to the
// canonical /leaderboard/export path and performs a strict host allowlist check.
func normalizeUDiscExportURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("empty URL")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL")
	}

	// Security: allowlist host
	if !strings.EqualFold(u.Host, "udisc.com") {
		return "", fmt.Errorf("unsupported host: %s", u.Host)
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
