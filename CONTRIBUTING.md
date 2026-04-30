# Contributing to GoBoard

Thanks for contributing to GoBoard.

## Quick flow

1. Fork repository
2. Create branch:
   ```bash
   git checkout -b feature/my-change
   ```
3. Implement change and add/update tests
4. Run checks locally
5. Open PR with clear description

## Local checks

```bash
# dashboard-api (root module)
go test ./...

# dashboard (library module)
(cd dashboard && go test ./...)
```

Optional coverage:

```bash
(cd dashboard && go test -coverprofile=../coverage.out -covermode=atomic ./...)
```

## Pull request checklist

- [ ] Change is focused and minimal
- [ ] Tests were added/updated if behavior changed
- [ ] `README.md` updated if API/runtime behavior changed
- [ ] No secrets or credentials included

## Code style

- Follow standard Go style (`gofmt`)
- Prefer small, readable functions
- Keep package APIs explicit and documented

## Reporting issues

When filing an issue, include:
- expected behavior
- actual behavior
- reproducible steps
- environment details (Go version, OS, Redis version)
