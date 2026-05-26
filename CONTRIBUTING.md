# Contributing

Keep changes small and behavior-preserving unless the issue explicitly calls
for a behavior change.

## Local Checks

Run before opening a pull request:

```sh
gofmt -w .
go test ./...
```

If topology generation changes, regenerate fixtures and rerun tests:

```sh
python testdata/generate_fixtures.py
go test ./...
```

## Behavioral Rule

Python FIM is the reference for topology semantics and GraphML compatibility.
When Go style and Python FIM behavior disagree, preserve Python FIM behavior.

Do not edit fixtures just to make a failing test pass. If fixture parity fails,
fix the Go implementation or intentionally regenerate fixtures from Python FIM
when the topology pattern itself changed.

## Package Boundaries

- `pkg/topology` is the primary consumer API.
- `pkg/catalog` owns catalog data and lookup.
- `pkg/sliver` owns ASM/FIM typed records and enum values.
- `pkg/diff` owns semantic graph comparison.
- `internal/graph` and `internal/graphml` are implementation details.

Provider code should not import internal packages.

## Dependencies

Avoid new external dependencies. The current non-standard dependency is
`github.com/google/uuid`.

## Pull Requests

Pull requests should include:

- focused code changes
- updated package README sections when public APIs change
- regenerated fixtures when topology fixture behavior changes
- passing CI
