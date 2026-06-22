# Contributing

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

