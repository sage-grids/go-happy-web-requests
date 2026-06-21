package racing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/sage-grids/go-happy-web-requests/internal/models"
)

// defaultMaxResponseBytes caps how much of a target response is read into
// memory. Override with MAX_RESPONSE_BYTES.
const defaultMaxResponseBytes = 10 << 20 // 10 MiB

func maxResponseBytes() int64 {
	if v := os.Getenv("MAX_RESPONSE_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxResponseBytes
}

func RaceHTTP(ctx context.Context, targetURL string, proxies []string) (models.RaceResult, error) {
	if len(proxies) == 0 {
		return models.RaceResult{}, fmt.Errorf("no proxies provided")
	}

	maxBytes := maxResponseBytes()

	resultCh := make(chan models.RaceResult, 1)
	errCh := make(chan error, len(proxies))

	for _, proxy := range proxies {
		go func(p string) {
			proxyURL, err := url.Parse(p)
			if err != nil {
				errCh <- err
				return
			}

			client := &http.Client{
				Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
			}

			req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
			if err != nil {
				errCh <- err
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
				if err != nil {
					errCh <- err
					return
				}
				select {
				case resultCh <- models.RaceResult{Proxy: p, Content: string(body)}:
				case <-ctx.Done():
				}
			} else {
				errCh <- fmt.Errorf("status %d from %s", resp.StatusCode, p)
			}
		}(proxy)
	}

	errors := 0
	for {
		select {
		case res := <-resultCh:
			return res, nil
		case <-errCh:
			errors++
			if errors == len(proxies) {
				return models.RaceResult{}, fmt.Errorf("all proxies failed")
			}
		case <-ctx.Done():
			return models.RaceResult{}, ctx.Err()
		}
	}
}
