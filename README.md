# Agent Doctor by NexoToken

**A local reliability and guidance layer for AI coding agents.**

Agent Doctor is an open-source local tool by [NexoToken](https://www.nexotoken.net/official), an AI API and agent tools platform for developers.

Codex and Claude Code do the work. Agent Doctor watches the evidence, detects
when a task is looping, losing context, or trying to finish without validation,
and sends a bounded next-step instruction back to the running agent. It also
explains why a task became slow, expensive, repetitive, or unreliable. Missing
data stays **unavailable** instead of becoming a fabricated zero.

The guidance engine is deterministic, runs locally, and does not call another
model. Raw prompts, source files, commands, tool inputs, and tool results are not
used by the guidance rules; supported hooks retain only bounded labels and
non-reversible evidence fingerprints.

## 60-second local start

Clone the repository and run one command. It installs dependencies, builds and
tests the product, installs the binary, configures the owned Codex and Claude
Code assets, opens the dashboard, and keeps the local service running:

```bash
./scripts/install-local.sh
```

`start` checks and idempotently prepares the Agent Doctor-owned Codex integration,
refreshes detected clients, starts the loopback services, and opens the dashboard
in the default browser. Use `start --no-open` when you want to copy the printed
URL yourself. Agent Doctor never binds a public interface. The dashboard includes
Overview, Task evidence, Costs, Memory, Comparison, Trends, Integrations, and
Privacy.

The simplest live capture path reuses the API base URL already configured in
your terminal and injects the local proxy only into the child process:

```bash
agent-doctor start                              # check integrations and open dashboard
agent-doctor run -- codex                       # start Codex with live capture
# or: agent-doctor run -- claude
```

`AGENT_DOCTOR_UPSTREAM_URL` has priority when a client does not expose its
configured base URL through `OPENAI_BASE_URL` or `ANTHROPIC_BASE_URL`. A client
that was already running must be restarted through the wrapper; Agent Doctor
does not attach to arbitrary existing processes or rewrite credentials.

After v1.0.0 is published, the verified installers will be:

```bash
curl --fail --proto '=https' --tlsv1.2 https://raw.githubusercontent.com/18534516725/Agent-Doctor/main/install.sh | sh
```

```powershell
irm https://raw.githubusercontent.com/18534516725/Agent-Doctor/main/install.ps1 | iex
```

Both installers download only the official GitHub release, verify its SHA-256,
install the binary, and run the consented setup step. See [installation and
uninstallation](docs/install.md) before using a remote installer.

## First diagnosis

```bash
agent-doctor doctor --json
agent-doctor diagnose --json
agent-doctor costs --json
agent-doctor dashboard --no-open
```

The first dashboard panel is the Task Guardian: it shows whether work is on
track, needs redirection, is waiting for validation, or has been blocked by a
capable hook. The result states evidence provenance and precision. A diagnosis
is not a verdict: unsupported evidence and absent billing records remain visible
limitations.

## Runtime guidance boundary

- **Claude Code:** official hooks can return guidance and, in `guard` or
  `autopilot`, enforce supported `PreToolUse` or unverified `Stop` decisions.
- **Codex:** MCP and Skill assets provide evidence-backed guidance at task
  checkpoints. The client can ignore MCP/Skill text, so this is not a
  deterministic block.
- **Other clients:** capability depends on the public interface listed in the
  compatibility matrix. Agent Doctor never claims enforcement where only
  observation or MCP advice exists.

Project control levels are `observe` (record only), `guide` (advise), `guard`
(enforce supported high-confidence rules), and `autopilot` (strongest supported
local controls). The default is `guide`.

## Public commands

| Command | Purpose |
| --- | --- |
| `agent-doctor setup --json` | Preview detected clients and exact owned changes |
| `agent-doctor setup --yes --json` | Apply the reviewed Codex MCP block |
| `agent-doctor setup --all --yes --json` | Install owned Codex MCP/Skill/AGENTS and Claude Hook/Skill assets |
| `agent-doctor start` | Check managed integrations, start loopback services, and open the dashboard |
| `agent-doctor start --no-open` | Same startup flow without opening a browser |
| `agent-doctor dashboard` | Alias for the visual workspace |
| `agent-doctor diagnose --json` | Summarize available task evidence |
| `agent-doctor compare --json` | Report matched-cohort readiness/results |
| `agent-doctor context --json` | Report bounded memory state without content |
| `agent-doctor costs --json` | Separate exact, estimated, and unavailable cost |
| `agent-doctor doctor --json` | Check database and detected clients |
| `agent-doctor pause --json` | Persistently pause local lifecycle capture |
| `agent-doctor pause --resume --json` | Resume local lifecycle capture |
| `agent-doctor export --json` | Export sanitized aggregates, never event payloads |
| `agent-doctor forget --yes --json` | Delete the local Agent Doctor database |
| `agent-doctor run -- <command>` | Start a client through the local capture proxy without shell evaluation |
| `agent-doctor uninstall --yes --json` | Remove only Agent Doctor-owned Codex config |
| `agent-doctor version` | Print the installed version |

## Supported clients

Codex, Claude Code, Cline, OpenCode, Cursor, Windsurf, Roo Code, Continue,
Aider, Cherry Studio, and generic command-line tools have declared capability
contracts. “Supported” does not mean that a client exposes every signal. See the
[compatibility matrix](docs/compatibility.md) for exact A/B/C capability levels
and installation boundaries.

## Cost truthfulness

- **exact**: a charge reported by a compatible billing source;
- **estimated**: local token usage multiplied by a versioned public catalog;
- **unavailable**: the required evidence does not exist.

These values are never merged into a misleading total. Currency conversion
requires a versioned rate. Read the [cost methodology](docs/cost-methodology.md).

## Local privacy

The SQLite database lives in the current user's configuration directory with
user-only permissions. When live capture is enabled, complete user, assistant,
system, and tool messages are stored locally so the owner can inspect the real
conversation. API keys, Authorization headers, cookies, and transport headers
are forwarded in memory and never written to SQLite. Replay is disabled until
the user approves the exact hashed plan, base commit, commands, call limit, and
cost limit. Read the [privacy model](docs/privacy.md).

## Architecture

```text
documented client interface → sanitizer → local event contract → SQLite
           ↑                                         ↓
   Hook / MCP / Skill ← deterministic guidance ← evidence fingerprints
                                                     ↓
                         safe aggregates → loopback Task Guardian
```

## Limitations

- Automatic setup currently owns only a marked Codex MCP block. Other tested
  adapters are shipped as explicit templates/plugins so user configuration is
  never silently rewritten.
- Comparisons require at least 15 matched samples per cohort and never declare
  an automatic winner.
- Agent Doctor does not read private client databases or bypass client
  permissions.
- No public release or ranking claim is valid until the tagged GitHub workflow
  completes and the published checksums are independently verified.

## Documentation

- [Install](docs/install.md)
- [Privacy](docs/privacy.md)
- [Cost methodology](docs/cost-methodology.md)
- [Diagnosis methodology](docs/diagnosis-methodology.md)
- [Compatibility](docs/compatibility.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Chinese usage guide](docs/usage.md)

## License

Apache License 2.0. See [LICENSE](LICENSE).
