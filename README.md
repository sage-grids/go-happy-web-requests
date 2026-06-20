# go-happy-web-requests

High-performance Go microservice for concurrent web scraping using the **Happy Eyeballs** pattern. Provide a target URL and a list of proxies — the service races them in parallel and returns the first successful response, canceling all slower requests.

## Quick Start

```bash
# Run locally
go run .

# Or with Docker (pre-built image)
docker run -d -p 8080:8080 -e API_TOKEN=your_token iserter/go-happy-web-requests:latest

# Or build from source
cp .env.example .env
docker compose up --build
```

## Deployment

### Coolify v4.1

1. Import this repo in Coolify
2. Set environment variables in Coolify UI:
   - `API_TOKEN` — required
   - `PORT` — optional (default: 8080)
3. Deploy

### Docker Compose

```bash
cp .env.example .env
# Edit .env with your values
docker compose up -d
```

### Docker Run

```bash
docker run -d \
  -p 8080:8080 \
  -e API_TOKEN=your_secret_token \
  -e PORT=8080 \
  iserter/go-happy-web-requests:0.1.0
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PORT` | `8080` | Server listen port |
| `API_TOKEN` | _(empty)_ | Bearer token for auth. If unset, auth is disabled |

## API

Full spec: [`docs/openapi.yaml`](docs/openapi.yaml) (OpenAPI 3.0)

### `POST /api/v1/fetch`

**Headers:** `Authorization: Bearer <token>`

**Request:**

```json
{
  "url": "https://example.com/data",
  "proxies": [
    "http://user:pass@proxy1.com:8080",
    "http://user:pass@proxy2.com:8080"
  ],
  "mode": "http",
  "timeout_seconds": 15
}
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `url` | yes | — | Target URL to fetch |
| `proxies` | yes | — | List of proxy URLs to race |
| `mode` | no | `http` | `http` or `playwright` (not yet implemented) |
| `timeout_seconds` | no | `15` | Max seconds before all requests are canceled |

**Success (200):**

```json
{
  "status": "success",
  "winning_proxy": "http://user:pass@proxy2.com:8080",
  "content": "<!DOCTYPE html>...",
  "time_taken_ms": 450
}
```

**Error (500):**

```json
{
  "status": "error",
  "message": "All proxies failed or timed out."
}
```

### cURL Example

```bash
curl -X POST http://localhost:8080/api/v1/fetch \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "proxies": ["http://proxy1:8080", "http://proxy2:8080"],
    "mode": "http",
    "timeout_seconds": 10
  }'
```

## Project Structure

```
├── main.go                      # Server entry point
├── internal/
│   ├── models/models.go         # Request/response types
│   ├── racing/http.go           # Happy Eyeballs racing engine
│   ├── middleware/auth.go       # Bearer token auth
│   └── handler/fetch.go         # API handler
├── docs/
│   ├── PRD.md                   # Product requirements
│   ├── openapi.yaml             # OpenAPI 3.0 spec
│   └── dev-guide/docker.md      # Docker publishing guide
├── .env.example                 # Environment variables template
├── Dockerfile
└── docker-compose.yml
```

## Testing

```bash
go test ./... -v -cover
```

## License

MIT
