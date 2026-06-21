# Dockerfile hardening

**Phase:** 2 — Production hardening
**Priority:** Medium
**Severity from review:** Low (#14)

## Problem
- The runtime image runs as `root` (`WORKDIR /root/`).
- Base images are unpinned (`golang:1.25-alpine`, `alpine:latest`).
- `COPY go.mod ./` omits `go.sum`; harmless today (no deps) but will silently
  break reproducible builds the moment a dependency is added.

## Proposed change
- Add a non-root user (or build to `FROM scratch` since the binary is static /
  `CGO_ENABLED=0`, copying CA certs + a passwd entry).
- Pin base images by digest (or at least a specific patch tag).
- `COPY go.mod go.sum ./` once dependencies exist; until then leave a comment.
- Confirm the compose healthcheck binary (`wget`) exists in the chosen final
  image — `scratch` has none, so either keep alpine or switch the healthcheck to
  the Go binary's own `-healthcheck` flag / `HEALTHCHECK` using the app.

## Acceptance criteria
- [ ] Container process runs as non-root.
- [ ] Base images pinned.
- [ ] Image builds and `docker compose up` reports `healthy`.
- [ ] go.sum handling documented/wired for future deps.
