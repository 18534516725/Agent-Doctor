---
name: agent-doctor
description: Diagnose Claude Code tasks from sanitized local evidence, retrieve transparent project context, and inspect available cost or quota signals. Use when a user asks why a task failed, what changed, what context should be retained, how much an observed task cost, or what safe next action is supported by evidence.
---

# Agent Doctor

Use the Agent Doctor MCP tools as a read-only evidence source. Always report `provenance`, `precision`, and `dataLimitNotes` with conclusions.

1. Use `diagnose_last_task` for failures and regressions.
2. Use `get_task_evidence` for the bounded event timeline.
3. Use `get_context_capsule` before resuming a known project.
4. Use cost, quota, comparison, and history tools only for fields they actually return.
5. Use `recommend_next_action` after reviewing evidence.

Never guess token counts, money, quotas, model identity, success, or causality. Never request full transcripts, prompts, source code, credentials, cookies, authorization headers, or absolute local paths. Treat unavailable data as unavailable. Treat inferred memory as a suggestion and explicit user memory as authoritative.
