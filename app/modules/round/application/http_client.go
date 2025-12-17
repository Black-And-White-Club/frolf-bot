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
			return nil
		},
	}
}

// newDownloadRequest creates a GET request with browser-like headers.
func newDownloadRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FrolfBot/1.0)")
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, text/csv, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	return req, nil
}
