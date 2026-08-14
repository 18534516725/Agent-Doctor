# Compatibility matrix

Capability levels describe verified interfaces, not marketing equivalence.

| Client | Level | Verified path | Important limit |
| --- | --- | --- | --- |
| Codex | A | MCP guidance + Skill + wrapper | guidance can be ignored by the client; no deterministic block |
| Claude Code | A | official Hook guidance/guard + MCP/Skill | enforcement only for documented Hook response points and `guard`/`autopilot` |
| Cline | A | official hook scripts, fail-open observation | feedback enforcement is not implemented |
| OpenCode | A | TypeScript plugin lifecycle adapter | plugin packaging must be installed explicitly |
| Cursor | B | MCP config + project rule | private editor history is not read |
| Windsurf | B | MCP config | lifecycle detail depends on exposed client events |
| Roo Code | B | MCP and hook fixtures | capability varies by extension version |
| Continue | B | MCP YAML template | no unsupported transcript extraction |
| Aider | C | generic argv-safe wrapper | only wrapper-observable evidence exists |
| Cherry Studio | C | documented configuration boundary | no private local database access |
| Generic CLI | C | `agent-doctor run -- ...` | no shell evaluation or hidden command capture |

Contract tests live beside every adapter. Missing evidence is reported as
unavailable. A lower level means a narrower verified interface, not lower model
quality.

“Guidance” and “enforcement” are deliberately separate capabilities. MCP,
Skills, and project rules can steer an agent but cannot guarantee compliance.
Only a client interface that accepts a blocking response can enforce a decision,
and Agent Doctor remains fail-open on storage errors, timeouts, unsupported
events, `observe`, healthy `continue`, or expired guidance.
