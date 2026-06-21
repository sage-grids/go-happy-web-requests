# Proxy retry / batching

**Phase:** 4 — Roadmap (PRD §6.2)
**Priority:** Low
**Severity from review:** Enhancement

## Problem
`RaceHTTP` races *all* provided proxies at once. The PRD envisions racing in
batches (e.g. batches of 3 out of 10): try a batch, and only if it fully fails,
feed in the next batch before giving up. This conserves proxy bandwidth and
upstream cost.

## Proposed change
- Add optional `batch_size` to the request (and/or a server default).
- Race in sequential batches: succeed early on first 2xx; on whole-batch failure
  advance to the next batch until exhausted or the request deadline hits.
- Preserve cancellation of in-flight losers within a batch.

## Acceptance criteria
- [ ] Batched racing implemented behind a config knob; default preserves current
      "race all" behavior unless set.
- [ ] Deadline still bounds total work across batches.
- [ ] Tests cover: batch 1 wins, batch 1 fails → batch 2 wins, all batches fail.

## Notes
Interacts with the `MAX_PROXIES` cap added in Phase 1.
