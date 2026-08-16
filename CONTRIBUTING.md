# Contributing

Agent Doctor accepts focused, tested changes that preserve its local-first and
evidence-first behavior.

## Requirements

- Go 1.25 or newer
- Node.js 22 or newer
- pnpm 10 or newer

## Contribution rules

1. Open an issue describing the user-visible problem and supported client.
2. Never attach real prompts, source code, credentials, cookies, or full logs.
3. Add a failing test before implementation.
4. Keep client adapters thin; diagnostic logic belongs in the Go core.
5. Treat missing telemetry as unavailable rather than inferred.
6. Run `git diff --check` before committing.

## Development checks

```bash
pnpm install --frozen-lockfile
go test ./... -race -count=1
pnpm --dir dashboard test -- --run
pnpm --dir dashboard build
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
```

Use the issue chooser before opening a pull request. Security vulnerabilities
must be reported privately according to `SECURITY.md`.
