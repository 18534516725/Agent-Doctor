---
name: agent-doctor
description: Keep Claude Code tasks on track with sanitized runtime guidance, diagnose failures, retrieve transparent project context, and inspect available cost or quota signals. Use at task start, after repeated failures or context compaction, before completion, and when the user asks for a diagnosis.
---

# Agent Doctor

Use the Agent Doctor MCP tools as a read-only evidence source. Always report `provenance`, `precision`, and `dataLimitNotes` with conclusions.

1. Call `get_runtime_guidance` at task start, after two similar failures, after context compaction, and before claiming completion.
2. Follow evidence-supported `redirect` and `verify` guidance. Treat `continue` as silence and add no diagnostic noise.
3. Use `diagnose_last_task` for failures and regressions.
4. Use `get_task_evidence` for the bounded event timeline.
5. Use `get_context_capsule` before resuming a known project.
6. Use cost, quota, comparison, and history tools only for fields they actually return.
7. Use `recommend_next_action` after reviewing evidence.

Never guess token counts, money, quotas, model identity, success, or causality. Never request full transcripts, prompts, source code, credentials, cookies, authorization headers, or absolute local paths. Treat unavailable data as unavailable. Treat inferred memory as a suggestion and explicit user memory as authoritative. The Skill/MCP path provides advice; never claim it blocked an action unless the active hook explicitly returned an enforcement decision.
