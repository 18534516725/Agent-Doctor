# Contributing

Agent Doctor accepts focused, tested changes that preserve its local-first and
evidence-first behavior.

## Requirements

- Go 1.25 or newer
- Node.js 24 or newer
- pnpm 10 or newer

## Contribution rules

1. Open an issue describing the user-visible problem and supported client.
2. Never attach real prompts, source code, credentials, cookies, or full logs.
3. Add a failing test before implementation.
4. Keep client adapters thin; diagnostic logic belongs in the Go core.
5. Treat missing telemetry as unavailable rather than inferred.
6. Run `git diff --check` before committing.

Detailed development commands will be added when the core workspaces land.
