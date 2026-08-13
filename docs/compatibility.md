# Compatibility matrix

Capability levels describe verified interfaces, not marketing equivalence.

| Client | Level | Verified path | Important limit |
| --- | --- | --- | --- |
| Codex | A | MCP + wrapper + project rules | setup auto-owns only its marked MCP block |
| Claude Code | A | official lifecycle hooks + MCP/skill assets | user installs the shipped plugin assets |
| Cline | A | official hook scripts, fail-open | only fields in the normalized contract persist |
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
