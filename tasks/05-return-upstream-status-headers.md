# Return upstream status & headers

**Phase:** 3 — Functional gaps
**Priority:** Medium
**Severity from review:** Medium (#10)

## Problem
The response returns `content` but not the upstream HTTP status code or
content-type. Non-2xx responses are collapsed into a generic "all proxies
failed", so the caller can't distinguish a 404 from a 503 from a dead proxy, and
can't tell HTML from JSON without sniffing.

## Proposed change
- Add `status_code int` and `content_type string` (and optionally selected
  response headers) to `models.RaceResult` / `models.FetchResponse`.
- Decide the contract for non-2xx: either
  (a) treat any response that completes as a "win" and surface its status, or
  (b) keep racing for a 2xx but report the last non-2xx status when all fail.
  Document the choice.

## Acceptance criteria
- [ ] Successful response includes `status_code` and `content_type`.
- [ ] Behavior for non-2xx is defined and tested.
- [ ] `time_taken_ms` no longer uses `omitempty` (review #12) so a 0ms value is
      still emitted.

## Notes
Folds in the small review nit #12 (`omitempty` on `time_taken_ms`).
