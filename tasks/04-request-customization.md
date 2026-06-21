# Request customization (method, headers, body, User-Agent)

**Phase:** 3 — Functional gaps
**Priority:** High
**Severity from review:** Medium (#9)

## Problem
Outbound requests are hardcoded `GET` with Go's default `User-Agent`
(`Go-http-client/1.1`) and no custom headers. Many target sites block the
default UA outright, so the service fails on real-world scraping targets. There
is also no way to scrape POST endpoints or send cookies/auth headers.

## Proposed change
Extend `models.FetchRequest`:
- `method` (default `GET`)
- `headers` (`map[string]string`)
- `body` (string, for non-GET)
- A sensible default `User-Agent` (browser-like), overridable via `headers`.

Thread these through `racing.RaceHTTP` into each `http.NewRequestWithContext`.

## Acceptance criteria
- [ ] Caller can set method, headers, body.
- [ ] A non-empty default User-Agent is sent when the caller doesn't supply one.
- [ ] Header/method values are validated (reject obviously malformed input).
- [ ] Tests cover custom headers and a POST with body.

## Notes
Coordinate with [09-ssrf-guardrails.md](09-ssrf-guardrails.md): arbitrary
methods + headers widen the SSRF surface.
