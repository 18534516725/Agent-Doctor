---
name: agent-doctor
description: Keep coding-agent tasks on track with sanitized, evidence-backed runtime guidance; diagnose failures, retrieve transparent project context, and inspect available cost or quota signals. Use at task start, after repeated failures or context compaction, before completion, and when the user asks for a diagnosis.
---

# Agent Doctor

Use the `agent-doctor` MCP tools to answer from local, sanitized evidence. Treat every result's `provenance`, `precision`, and `dataLimitNotes` as part of the answer.

## Workflow

1. Call `get_runtime_guidance` at task start. Call it again after two similar failures, after context compaction, and before claiming completion.
2. Follow an evidence-supported `redirect` or `verify` instruction. Treat `continue` as silence and do not add diagnostic noise.
3. For a captured coding task, call `get_project_analysis` before the final answer when broader health, efficiency, or cost evidence is relevant.
4. Identify the task or project from the user's request. Ask for a session or project identifier only when the local tools cannot resolve it.
5. Start with the narrowest additional read-only tool when deeper evidence is relevant:
   - `diagnose_last_task` for failures, regressions, or unexpected outcomes.
   - `get_task_evidence` for the bounded evidence timeline.
   - `get_context_capsule` before resuming work in a known project.
   - `get_cost_summary` or `get_quota_status` for usage questions.
   - `compare_clients`, `compare_models`, or `get_performance_history` for observed comparisons.
   - `recommend_next_action` only after reviewing the evidence.
6. Separate exact observations, estimates, and unavailable data in the response.
7. Explain limitations and recommend only reversible or user-approved validation steps.

## Safety boundaries

- Keep the workflow read-only unless the user separately authorizes an action outside Agent Doctor.
- MCP and Skill integrations provide guidance only. Never claim Agent Doctor blocked an action unless the connected client explicitly reports enforcement capability.
- Never guess missing token counts, money, quotas, model identity, success, or causality.
- Never request or reproduce prompts, source code, credentials, cookies, authorization headers, or absolute local paths.
- Do not scrape another application's private storage. Use only Agent Doctor's normalized evidence.
- If a tool reports unavailable evidence, say that directly and explain what compatible signal would be needed.
- Treat inferred memory as a suggestion, not as a user preference. Explicit user memory takes precedence.

## Response format

Lead with the diagnosis or answer. Then state the strongest evidence, its precision, its provenance, and any data limits. End with one safe next action when the evidence supports it.
