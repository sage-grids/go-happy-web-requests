# Product Requirements Document (PRD)

**Project Name:** `sage-grids/go-happy-web-requests`
**Version:** 1.0.0
**License:** MIT

## 1. Overview & Purpose

`go-happy-web-requests` is a high-performance Go microservice designed for robust web scraping. Utilizing the "Happy Eyeballs" concurrency pattern, the service accepts a target URL and a list of proxies, fires requests in parallel, and instantly returns the first successful response. Slower or failing requests are immediately canceled via Go Contexts to conserve network and CPU resources.

The application exposes a REST API for easy integration with existing architectures and supports both standard HTTP requests and headless browser automation (Playwright).

## 2. Goals & Non-Goals

### Goals

* **Speed & Reliability:** Guarantee the fastest possible response time by racing multiple proxies simultaneously.
* **Resource Efficiency:** Instantly terminate losing requests to prevent memory leaks and wasted bandwidth.
* **Dual-Mode Operation:** Support fast programmatic HTTP requests and full DOM-rendering via Playwright.
* **Simple Integration:** Provide a stateless REST API secured by a simple Bearer token.
* **Containerized:** Provide out-of-the-box Docker support for easy deployment and scaling.

### Non-Goals

* **Proxy Storage:** The service will not act as a database for proxy health. Proxies must be provided by the client per request.
* **Scraping Logic:** The service returns raw HTML/JSON; it will not parse or extract specific DOM elements (e.g., Beautifulsoup/Cheerio logic).

## 3. API Specification

### Authentication

All API requests must include an `Authorization` header. The token must match the `API_TOKEN` environment variable set on the server.
`Authorization: Bearer <YOUR_API_TOKEN>`

### `POST /api/v1/fetch`

Initiates a concurrent fetch request.

**Request Body (JSON):**

```json
{
  "url": "https://example.com/data",
  "proxies": [
    "http://user:pass@proxy1.com:8080",
    "http://user:pass@proxy2.com:8080",
    "http://user:pass@proxy3.com:8080"
  ],
  "mode": "http", // "http" or "playwright"
  "timeout_seconds": 15
}

```

**Response (200 OK):**

```json
{
  "status": "success",
  "winning_proxy": "http://user:pass@proxy2.com:8080",
  "content": "<!DOCTYPE html><html>...</html>",
  "time_taken_ms": 450
}

```

**Response (500 Internal Server Error):**

```json
{
  "status": "error",
  "message": "All proxies failed or timed out."
}

```

## 4. Sample Core Implementation (Go)

Below is the foundational code for `main.go`. It implements the HTTP server, the Bearer token middleware, and the core Go-routine racing logic.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// --- Models ---

type FetchRequest struct {
	URL            string   `json:"url"`
	Proxies        []string `json:"proxies"`
	Mode           string   `json:"mode"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type FetchResponse struct {
	Status       string `json:"status"`
	WinningProxy string `json:"winning_proxy,omitempty"`
	Content      string `json:"content,omitempty"`
	TimeTakenMs  int64  `json:"time_taken_ms,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Result struct for channels
type RaceResult struct {
	Proxy   string
	Content string
}

// --- Racing Engine ---

func RaceHTTP(ctx context.Context, targetURL string, proxies []string) (RaceResult, error) {
	resultCh := make(chan RaceResult, 1)
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

			req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
			resp, err := client.Do(req)

			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				body, _ := io.ReadAll(resp.Body)
				select {
				case resultCh <- RaceResult{Proxy: p, Content: string(body)}:
				case <-ctx.Done(): // Lost the race
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
				return RaceResult{}, fmt.Errorf("all proxies failed")
			}
		case <-ctx.Done():
			return RaceResult{}, ctx.Err()
		}
	}
}

// --- API Handlers & Middleware ---

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("API_TOKEN")
		if expectedToken == "" {
			// If no token is configured, allow traffic (or block depending on preference)
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != expectedToken {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func FetchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var reqBody FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Default timeout
	timeout := 15
	if reqBody.TimeoutSeconds > 0 {
		timeout = reqBody.TimeoutSeconds
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	var res RaceResult
	var err error

	if reqBody.Mode == "playwright" {
		// NOTE: Implement Playwright racing logic here
		err = fmt.Errorf("playwright mode not yet implemented in this sample")
	} else {
		res, err = RaceHTTP(ctx, reqBody.URL, reqBody.Proxies)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FetchResponse{Status: "error", Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(FetchResponse{
		Status:       "success",
		WinningProxy: res.Proxy,
		Content:      res.Content,
		TimeTakenMs:  time.Since(start).Milliseconds(),
	})
}

func main() {
	http.HandleFunc("/api/v1/fetch", AuthMiddleware(FetchHandler))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

```

## 5. Deployment Setup

### `Dockerfile`

A multi-stage build ensuring a tiny, secure, and fast Docker image. *(Note: If you fully implement Playwright, you will need to use a base image that includes Playwright dependencies instead of standard alpine).*

```dockerfile
# Build Stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o go-happy-web-requests .

# Run Stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/go-happy-web-requests .

EXPOSE 8080
CMD ["./go-happy-web-requests"]

```

### `docker-compose.yml`

Simplifies deployment and allows easy environment variable injection.

```yaml
version: '3.8'

services:
  go-happy-web-requests:
    build: .
    container_name: go-happy-web-requests
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - API_TOKEN=super_secret_token_123 # Change this in production!

```

## 6. Next Steps & Future Enhancements

1. **Playwright Integration:** Implement the `playwright-go` library within the `reqBody.Mode == "playwright"` block using the exact same `context` and `channel` pattern shown for `RaceHTTP`.
2. **Retry Queue:** Add logic that slices an array of e.g., 10 proxies into batches of 3. If the first batch fails, the system automatically loops and feeds the next batch of 3 into the racing engine before giving up.
3. **Metrics:** Add Prometheus metrics to track average latency, proxy failure rates, and success ratios to help you prune bad proxies from your upstream provider.
