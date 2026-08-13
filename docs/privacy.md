# Local privacy and data controls

Agent Doctor is local-first. The daemon listens only on a random loopback
address, requires a fresh 256-bit session token for APIs, rejects foreign
origins, denies framing, sends `no-referrer`, and does not load remote dashboard
assets.

## Not captured by default

- complete prompts and transcripts;
- source or file contents;
- command arguments and working-directory paths;
- API keys, authorization headers, cookies, or credentials;
- internal providers, routing channels, or raw upstream errors.

Events pass through a bounded JSON contract and privacy filter before SQLite.
Dashboard snapshot responses select aggregates and safe labels only. Session
evidence returns event ID, time, type, provenance, and precision—not payloads.

## Controls

The Privacy dashboard controls prompt/file-content flags and retention period.
Both sensitive flags default to off. `agent-doctor export --json` exports safe
aggregates. `agent-doctor forget --yes --json` removes only the local database
and its SQLite sidecar files.

## Controlled execution

Validation commands must exactly match an in-memory allowlist and never pass
through a shell evaluator. Replay additionally requires one-time consent bound
to the exact plan hash, a resolvable base SHA, approved argv commands, maximum
calls, maximum cost, and deterministic cleanup of a detached external worktree.

## Network boundary

Core diagnosis works offline. NexoToken usage import and update checking are
explicit optional network actions. Tokens used for an import are not stored in
diagnostic events or logs.
