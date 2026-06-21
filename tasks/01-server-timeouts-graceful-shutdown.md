# Server timeouts & graceful shutdown

**Phase:** 2 — Production hardening
**Priority:** High
**Severity from review:** Medium (#8)

## Problem
`main.go` uses bare `http.ListenAndServe`. There are no server-level timeouts
(`ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`), leaving the
service exposed to slowloris-style connection holding. There is also no signal
handling, so in-flight requests are killed abruptly on deploy/restart.

## Proposed change
- Replace the bare listener with a configured `*http.Server`:
  - `ReadHeaderTimeout` (e.g. 5s), `ReadTimeout`, `IdleTimeout`.
  - Leave `WriteTimeout` generous or unset because the per-request context
    already bounds the racing work; document the interaction with
    `timeout_seconds`.
- Trap `SIGINT`/`SIGTERM` via `signal.NotifyContext` and call
  `srv.Shutdown(ctx)` with a bounded drain timeout.

## Acceptance criteria
- [ ] Server starts via `*http.Server` with explicit timeouts.
- [ ] SIGTERM drains in-flight requests, then exits 0.
- [ ] Timeouts are env-configurable with sane defaults.
- [ ] A test (or documented manual check) confirms a slow-header client is cut off.

## Notes
Keep the request-scoped `context.WithTimeout` in the handler; this task is about
the connection/server layer, not the per-fetch deadline.
