---
name: agent-doctor
description: Keep Claude Code tasks on track with sanitized runtime guidance, diagnose failures, retrieve transparent project context, and inspect available cost or quota signals. Use at task start, after repeated failures or context compaction, before completion, and when the user asks for a diagnosis.
---

# Agent Doctor

Use the Agent Doctor MCP tools as a local evidence source. Evidence calls do not modify the workspace; runtime guidance and context-capsule calls record only bounded local delivery receipts. Always report `provenance`, `precision`, and `dataLimitNotes` with conclusions.

At `SessionStart`, the installed hook automatically injects a bounded cross-client handoff for the same project when local evidence exists. Treat it as a resumable task snapshot, not as a complete transcript, and verify it against the current workspace.

1. At the start of every coding turn, call `get_runtime_guidance` with `projectId` set to the current project ID. Call it again after a failed tool step, after context compaction, and before claiming completion.
2. Follow evidence-supported `advise`, `redirect`, `verify`, and `block` guidance. Briefly tell the user `Agent Doctor intervened: <action taken>` when non-`continue` guidance changes the work. Treat `continue` as silence and add no diagnostic noise.
3. Use `diagnose_last_task` for failures and regressions.
4. Use `get_task_evidence` for the bounded event timeline.
5. Use `get_context_capsule` before resuming a known project.
6. Use cost, quota, comparison, and history tools only for fields they actually return.
7. Use `recommend_next_action` after reviewing evidence.

Never guess token counts, money, quotas, model identity, success, or causality. Never request full transcripts, prompts, source code, credentials, cookies, authorization headers, or absolute local paths. Treat unavailable data as unavailable. Treat inferred memory as a suggestion and explicit user memory as authoritative. The Skill/MCP path provides advice; never claim it blocked an action unless the active hook explicitly returned an enforcement decision.
