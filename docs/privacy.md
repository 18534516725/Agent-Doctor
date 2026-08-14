# Local privacy and data controls

Agent Doctor is local-first. The daemon listens only on a random loopback
address, requires a fresh 256-bit session token for APIs, rejects foreign
origins, denies framing, sends `no-referrer`, and does not load remote dashboard
assets.

## Local full-conversation mode

When a client is started through the loopback capture proxy, Agent Doctor stores
its complete user, assistant, system, and tool messages in the local SQLite
database. This is what powers the private conversation timeline. It never
uploads those messages and never exposes them through aggregate exports or MCP
evidence tools.

The following transport and machine secrets are never persisted:

- source or file contents;
- command arguments and working-directory paths;
- API keys, authorization headers, cookies, or credentials;
- internal providers, routing channels, or raw upstream errors.

Events pass through a bounded JSON contract and privacy filter before SQLite.
Dashboard snapshot responses select aggregates and safe labels only. Session
evidence returns event ID, time, type, provenance, and precision—not payloads.

## Runtime guidance data

Runtime guidance does not send task content to another model. Claude Code tool
input and result JSON are canonicalized in memory and immediately reduced to
SHA-256 fingerprints; the raw JSON is not copied into lifecycle events.
Deterministic rules operate on event IDs, timestamps, bounded tool labels,
progress/validation facts, and those non-reversible fingerprints. Persisted
guidance contains the finding, instruction, source event IDs, control metadata,
and expiry—not prompts, source code, commands, paths, tool inputs, or tool
results.

MCP/Skill clients receive sanitized guidance only. Claude Code Hook responses
are token-bounded, credential-filtered, path-filtered, and fail open: any local
error or timeout produces no stdout control response and Claude continues.

## Controls

The Privacy dashboard controls message/file-content flags and retention period.
Complete model messages default to on; file-content capture defaults to off.
These settings persist in SQLite. `agent-doctor export --json` exports safe
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
