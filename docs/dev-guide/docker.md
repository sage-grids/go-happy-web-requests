# Docker Publishing Guide

## Publishing to sage-grids Docker Hub

### Prerequisites

1. Docker Hub account with write access to the `sagegrids` organization
2. Docker CLI authenticated: `docker login`
3. Go 1.25+ installed locally

### Build & Push

```bash
# Build the image
docker build -t sagegrids/go-happy-web-requests:latest .

# Tag with version (optional but recommended)
docker tag sagegrids/go-happy-web-requests:latest sagegrids/go-happy-web-requests:1.0.0

# Push both tags
docker push sagegrids/go-happy-web-requests:latest
docker push sagegrids/go-happy-web-requests:1.0.0
```

### Multi-Architecture Build (amd64 + arm64)

```bash
docker buildx create --name multiarch --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t sagegrids/go-happy-web-requests:latest \
  -t sagegrids/go-happy-web-requests:1.0.0 \
  --push .
```

### CI/CD (GitHub Actions)

Add to `.github/workflows/docker-publish.yml`:

```yaml
name: Docker Publish

on:
  release:
    types: [published]

jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Login to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            sagegrids/go-happy-web-requests:latest
            sagegrids/go-happy-web-requests:${{ github.ref_name }}
```

**Required secrets:**
- `DOCKERHUB_USERNAME` — Docker Hub username with org write access
- `DOCKERHUB_TOKEN` — Docker Hub access token (not password)

### Versioning Strategy

| Tag | When to use |
|-----|-------------|
| `latest` | Every release |
| `1.0.0` | Semantic version from git tag |
| `1.0` | Minor version (optional) |

### Deployment

```bash
docker run -d \
  --name go-happy-web-requests \
  -p 8080:8080 \
  -e API_TOKEN=your_secret_token \
  -e PORT=8080 \
  sagegrids/go-happy-web-requests:latest
```

Or with docker-compose:

```bash
docker compose up -d
```

### Verifying the Image

```bash
# Pull and inspect
docker pull sagegrids/go-happy-web-requests:latest
docker inspect sagegrids/go-happy-web-requests:latest

# Test locally
docker run --rm -p 8080:8080 -e API_TOKEN=test sagegrids/go-happy-web-requests:latest

# In another terminal
curl -X POST http://localhost:8080/api/v1/fetch \
  -H "Authorization: Bearer test" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","proxies":["http://proxy:8080"]}'
```
