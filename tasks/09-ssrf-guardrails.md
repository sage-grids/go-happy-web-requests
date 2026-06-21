# SSRF guardrails

**Phase:** Cross-cutting
**Priority:** High (deliberate design decision required)
**Severity from review:** High (#4)

## Problem
The service fetches arbitrary URLs through caller-supplied proxies and returns
the body, with zero guardrails. There is nothing preventing requests to
`http://169.254.169.254/` (cloud metadata), `localhost`, or RFC1918 ranges.
Some of this is inherent to a scraping proxy, but the *absence of any control or
documented stance* is the risk.

## Proposed change (decide the posture first)
- Add an optional target allow/deny policy:
  - deny-list of CIDRs/hosts (loopback, link-local, RFC1918, metadata IPs) on by
    default, with an opt-out env for trusted internal deployments;
  - or an explicit allow-list mode.
- Validate/normalize the target URL scheme (`http`/`https` only).
- Note: when proxies are used, DNS resolution happens at the proxy, so host
  checks must consider both direct and proxied resolution. Document the limits.

## Acceptance criteria
- [ ] Documented decision on the default posture (deny-internal vs open).
- [ ] Scheme validation on the target URL.
- [ ] Configurable deny/allow policy with tests.
- [ ] README/PRD updated to state the security stance explicitly.

## Notes
This is partly a product decision, not just code. Surface it to the owner before
implementing a default that could break legitimate internal-scraping use cases.
