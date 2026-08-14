# Install, update, and uninstall

## Requirements

- macOS, Linux, or Windows on amd64 or arm64;
- a local browser for the visual dashboard;
- Git only for controlled replay features;
- no cloud account is required for local diagnostics.

## Build locally now

```bash
pnpm install --frozen-lockfile
./scripts/embed-dashboard.sh
go build -trimpath -o ./bin/agent-doctor ./cmd/agent-doctor
./bin/agent-doctor doctor --json
./bin/agent-doctor start --once --no-open
```

`start --once --no-open` validates managed integration setup, database migration,
dashboard embedding, and loopback binding, then exits. For normal use, run only:

```bash
agent-doctor start
```

This checks the Agent Doctor-owned Codex integration, refreshes detected clients,
starts the local services, and opens the dashboard. `start --no-open` performs
the same checks but only prints the URL.

## Configure Codex safely

Ordinary startup performs this idempotent managed setup automatically. To inspect
the exact change without applying it, use:

```bash
agent-doctor setup --json
```

Apply the exact plan:

```bash
agent-doctor setup --yes --json
```

The installer preserves existing `~/.codex/config.toml` content, writes a
clearly marked `mcp_servers.agent_doctor` block atomically, saves a user-only
backup, and is idempotent. Other clients use the tested files under `adapters/`;
they are not silently edited by the current setup command.

## Uninstall and local data

```bash
agent-doctor uninstall --yes --json
```

This removes only the owned Codex block. It does not remove user settings or
diagnostic history. To explicitly delete local history:

```bash
agent-doctor forget --yes --json
```

These two actions are intentionally separate.

## Official release verification

Release archives are named
`agent-doctor_<version>_<os>_<arch>.tar.gz` (or `.zip` on Windows). Verify the
matching entry in `SHA256SUMS.txt` before extraction. The update engine also
requires the exact official GitHub release host/path, version, architecture,
file size, and SHA-256. It never terminates an AI client.
