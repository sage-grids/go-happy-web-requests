package racing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sage-grids/go-happy-web-requests/internal/models"
)

func RaceHTTP(ctx context.Context, targetURL string, proxies []string) (models.RaceResult, error) {
	if len(proxies) == 0 {
		return models.RaceResult{}, fmt.Errorf("no proxies provided")
	}

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
				body, err := io.ReadAll(resp.Body)
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
