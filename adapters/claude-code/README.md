# Agent Doctor for Claude Code

This plugin uses Claude Code's documented plugin, MCP, and hook contracts. It injects a bounded context capsule at `SessionStart` and records only allowlisted lifecycle metadata for diagnostics.

The hook commands use exec-form `args`, a 500 ms timeout, and exit code 0 on every collection failure. They do not make permission decisions, rewrite prompts, or read transcript files. Tool inputs and tool responses are discarded by the normalizer.

Reference: <https://code.claude.com/docs/en/hooks> and <https://code.claude.com/docs/en/plugins-reference>.
