# CI pipeline

**Phase:** 2 — Production hardening
**Priority:** Medium
**Severity from review:** Low (#15)

## Problem
There is no automated CI. Build, vet, tests, and lint only run locally.

## Proposed change
Add a GitHub Actions workflow (`.github/workflows/ci.yml`) that runs on push and
PR:
- `go build ./...`
- `go vet ./...`
- `go test -race -cover ./...`
- `golangci-lint run`
- (optional) `docker build` to validate the Dockerfile.

## Acceptance criteria
- [ ] Workflow runs on push + PR against `main`.
- [ ] Tests run with `-race`.
- [ ] Lint step present (golangci-lint).
- [ ] Status badge added to README.

## Notes
The race detector will exercise the goroutine fan-out in `racing` — worth having
given the concurrency model.
