# Test cleanup

**Phase:** Cross-cutting
**Priority:** Low
**Severity from review:** Low (#16, #17)

## Problems
1. `TestRaceHTTP_CancelsLosingGoroutines` (`internal/racing/http_test.go`) sleeps
   ~5s and its assertion only `t.Log`s — it can never fail. It dominates the
   package's ~7s runtime while verifying nothing.
2. Several middleware tests use `os.Setenv`/`os.Unsetenv` instead of
   `t.Setenv`, which auto-restores and guards against parallel runs. (Phase 1
   already converted some; finish the rest.)

## Proposed change
- Rewrite the cancellation test to make a real assertion (e.g. observe that the
  slow proxy's handler was cancelled), or shorten/remove it.
- Convert remaining `os.Setenv`/`os.Unsetenv` calls to `t.Setenv`.
- Consider `t.Parallel()` where safe once env usage is `t.Setenv`-based.

## Acceptance criteria
- [ ] No test relies on a `t.Log`-only "assertion".
- [ ] Racing package test time meaningfully reduced.
- [ ] Env mutation uses `t.Setenv` throughout.
