# Playwright mode

**Phase:** 4 — Roadmap (PRD §6.1)
**Priority:** Medium
**Severity from review:** Functional stub

## Problem
`mode: "playwright"` currently returns HTTP 500 "playwright mode not yet
implemented" (`handler/fetch.go`). The PRD lists full DOM rendering via
Playwright as a core goal.

## Proposed change
- Integrate `playwright-go`, racing browser contexts across proxies using the
  same `context` + channel pattern as `RaceHTTP`.
- Return rendered DOM as `content`.
- Each browser context must use its proxy and respect the request deadline;
  losing contexts must be cancelled/closed to avoid leaking browser processes.

## Acceptance criteria
- [ ] `mode: "playwright"` returns rendered HTML.
- [ ] Losing/timed-out browser contexts are reliably torn down (no zombie
      processes).
- [ ] Per-proxy racing semantics match HTTP mode.
- [ ] Tests cover success, all-fail, and timeout paths.

## Notes
The runtime image must include browser dependencies — the current
`alpine`/`scratch` image won't work. Coordinate with
[02-dockerfile-hardening.md](02-dockerfile-hardening.md); likely a separate
image variant. Memory limits (currently 256M) will need revisiting.
