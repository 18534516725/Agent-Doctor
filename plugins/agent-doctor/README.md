# Agent Doctor by NexoToken for Codex

This repo-local plugin exposes Agent Doctor's sanitized, read-only MCP tools and the `$agent-doctor` diagnostic skill to Codex.

## Runtime contract

- The `agent-doctor` binary must be available on `PATH`.
- Codex starts the local MCP server with `agent-doctor mcp serve`.
- MCP responses label provenance and precision; unavailable values are not inferred.
- The plugin does not read Codex private storage, prompts, source code, credentials, or full transcripts.

## Lifecycle integration status

The repository includes a fail-open Codex lifecycle normalizer in `internal/adapters/codex`. It accepts only supported lifecycle facts, hashes working-directory identity, and discards unknown fields. The plugin does not declare an undocumented hook manifest. Lifecycle automation will be enabled only when a supported public Codex hook contract can be validated; MCP and the skill work independently of that optional integration.

## Development validation

```bash
go test ./internal/adapters/codex ./plugins/agent-doctor ./internal/privacy
python3 "$CODEX_PLUGIN_CREATOR/scripts/validate_plugin.py" plugins/agent-doctor
python3 "$CODEX_SKILL_CREATOR/scripts/quick_validate.py" plugins/agent-doctor/skills/agent-doctor
```
