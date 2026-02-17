package roundservice

import (
	"context"
	"fmt"
	"net/http"
)

// newDownloadClient returns an *http.Client configured with sensible defaults
// for downloading scorecard exports (timeout and redirect limits).
func newDownloadClient() *http.Client {
	return &http.Client{
		Timeout: downloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			if err := validateUDiscURL(req.URL); err != nil {
				return fmt.Errorf("redirect target rejected: %w", err)
			}
			return nil
		},
	}
}

// newDownloadRequest creates a GET request with browser-like headers.
func newDownloadRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	u, err := parseAndValidateUDiscURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid download URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FrolfBot/1.0)")
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, text/csv, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	return req, nil
}
