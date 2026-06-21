# Structured logging & metrics

**Phase:** 3 — Functional gaps
**Priority:** Medium
**Severity from review:** Medium (#18) + PRD §6.3

## Problem
There is no request logging, no structured logs, and no metrics. Operating the
service in production (diagnosing failures, pruning bad proxies) is blind.

## Proposed change
- Adopt `log/slog` for structured logs; add a request-logging middleware
  (method, path, status, duration, request id).
- Expose Prometheus metrics on a separate endpoint (`/metrics`):
  - request count / latency histogram per outcome
  - per-proxy success/failure counters
  - racing win latency
- **Important:** never log full proxy URLs — they contain `user:pass`
  credentials. Redact credentials before logging.

## Acceptance criteria
- [ ] Structured request logs with duration + status.
- [ ] `/metrics` endpoint with the counters above.
- [ ] Proxy credentials redacted everywhere they could be logged.
- [ ] `/metrics` access policy decided (internal-only vs authenticated).

## Notes
Be careful that proxy credentials don't leak into the `winning_proxy` response
field or error messages either (review noted echoed proxy strings).
