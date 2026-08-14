# Agent Doctor

**Local live conversations, evidence, context memory, and cost analytics for AI coding agents.**

Agent Doctor explains why a coding task became slow, expensive, repetitive, or
unreliable. It combines complete model conversations captured through its
loopback proxy, lifecycle events, Git state, validations, usage, cost evidence,
and local history. Missing data stays **unavailable** instead of becoming a
fabricated zero.

## 60-second local start

Until the first signed release is published, build from this repository:

```bash
pnpm install --frozen-lockfile
./scripts/embed-dashboard.sh
go build -o ./bin/agent-doctor ./cmd/agent-doctor
./bin/agent-doctor setup --yes --json
./bin/agent-doctor start --no-open
```

Open the printed `http://127.0.0.1:<random-port>/` URL yourself. Agent Doctor
never binds a public interface. The dashboard includes Overview, Task evidence,
Costs, Memory, Comparison, Trends, Integrations, and Privacy.

The simplest live capture path reuses the API base URL already configured in
your terminal and injects the local proxy only into the child process:

```bash
agent-doctor start --no-open                    # keep the dashboard running
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

The result states evidence provenance and precision. A diagnosis is not a
verdict: unsupported evidence, small cohorts, and absent billing records remain
visible limitations.

## Public commands

| Command | Purpose |
| --- | --- |
| `agent-doctor setup --json` | Preview detected clients and exact owned changes |
| `agent-doctor setup --yes --json` | Apply the reviewed Codex MCP block |
| `agent-doctor start --no-open` | Start the authenticated loopback dashboard |
| `agent-doctor dashboard --no-open` | Alias for the visual workspace |
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
                                                     ↓
MCP read-only tools ← evidence engine ← safe aggregates → loopback dashboard
                                                     ↓
                              consent-bound detached-worktree replay
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
