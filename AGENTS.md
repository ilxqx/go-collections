# Agent instructions

## Testing

- Write assertions with `testify`: `assert` for a check the test should survive,
  `require` for one it cannot continue past. Do not hand-roll `if got != want {
  t.Errorf(...) }`.
- Map the two by what the native code did: a former `t.Errorf` becomes an
  `assert` call, a former `t.Fatalf` or `t.Fatal` becomes a `require` call.
  Setup that later assertions depend on — building a fixture, decoding
  serialized data, an `err` that makes the rest meaningless — is always
  `require`.
- A length or presence check that guards an index is `require`, and the fields
  it guards are `assert`: `require.Len(t, got, 2)` then `assert.Equal(t, 10,
  got[0])`.
- Compare an expected empty slice with `assert.Empty`, never `assert.Equal`
  against `nil`. `slices.Equal(nil, []T{})` is true but `assert.Equal` compares
  with `reflect.DeepEqual`, which separates a nil slice from an empty one, so
  `assert.Equal` would tighten the assertion without saying so.
- Use the typed helpers where they read better than a boolean: `assert.Len`,
  `assert.NoError`, `assert.ErrorIs`, `assert.Panics`, `assert.NotPanics`,
  `assert.True`. `assert.Equal` takes `(t, want, got)` in that order.
- Give every assertion a message when the call site alone does not say which
  case failed — the `f` variants (`assert.Equalf`) take a format string. A
  table-driven case that already names itself through `t.Run` does not need one.
- In a `testify/suite` test, call assertions on the suite receiver (`s.Equal`,
  `s.Require().Error`), not on the `assert`/`require` packages, and name the
  receiver `s` so it does not shadow the `suite` import.
- Keep `Example` functions and benchmarks in plain Go. An example is compiled
  documentation checked against its `// Output:` block, and a benchmark has
  nothing to assert.
- Concurrent containers get concurrent tests: exercise the atomic single-key
  operations under `-race` with real goroutines, and assert only what the
  documented atomicity guarantees — bulk operations are best-effort snapshots.

## Linting

- `task check` runs the formatter, `go vet`, the linter and the tests.
- The linter version is pinned in CI. Raise it deliberately, not incidentally.
- `testifylint` enforces the assert/require split and the empty-collection rule
  above, so most of the Testing section is checked rather than reviewed.
- Prefer a rule exception in `.golangci.yml`, carrying the reason, over a
  `//nolint` comment in the source.
- The `.golangci.yml` baseline (enabled linters and generic settings) is shared
  with coldsmirk/go-streams; keep the two in step when changing it. The rule
  exceptions and their reasons are per-project.
