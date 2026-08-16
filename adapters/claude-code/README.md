# Agent Doctor for Claude Code

This plugin uses Claude Code's documented plugin, MCP, and hook contracts. At `SessionStart` it resolves both the privacy-hashed working directory and the matching local project identity, then injects a maximum 800-token cross-client handoff built from confirmed project memory and the latest captured task snapshot.

The hook commands use exec-form `args`, a bounded timeout, and exit code 0 on every collection failure. The handoff does not copy a full transcript, rewrite the user prompt, or read Claude Code transcript files. Tool inputs and tool responses are discarded by the normalizer. Each successful injection records a local delivery receipt visible in the dashboard.

Reference: <https://code.claude.com/docs/en/hooks> and <https://code.claude.com/docs/en/plugins-reference>.
