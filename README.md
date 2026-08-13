# Agent Doctor

Local diagnostics, context memory, and cost analytics for AI coding agents.

Agent Doctor helps you understand why a real coding task became slow, expensive,
or unreliable by using local evidence from Git, approved validations, client
events, and your own historical baseline. By NexoToken.

## Project status

Agent Doctor is under active development. The first public release will provide
one-command installation, local-only storage, evidence-based task diagnostics,
context capsules, cost and quota analysis, client adapters, and a visual local
dashboard.

The repository does not currently publish an installable release. Do not use
unofficial binaries that claim to be Agent Doctor.

For the current capability matrix, local MCP verification, and data boundaries,
read [the usage guide](docs/usage.md).

## Privacy baseline

- Data stays on the user's device by default.
- Source code, complete prompts, credentials, cookies, and raw request headers
  are not stored by default.
- Project commands run only after the user explicitly approves the exact command.
- Missing model, token, or billing data remains unavailable; it is never guessed.

## License

Apache License 2.0. See [LICENSE](LICENSE).
