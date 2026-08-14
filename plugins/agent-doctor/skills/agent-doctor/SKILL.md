---
name: agent-doctor
description: Diagnose coding-agent tasks from sanitized local evidence, retrieve transparent project context, compare observed clients or public model names, and inspect available cost or quota signals. Use after completing a captured coding task and when a user asks why a task failed or became slow, what changed, what context should be carried forward, how much an observed task cost, or which safe next action is supported by evidence.
---

# Agent Doctor

Use the `agent-doctor` MCP tools to answer from local, sanitized evidence. Treat every result's `provenance`, `precision`, and `dataLimitNotes` as part of the answer.

## Workflow

1. For a captured coding task, call `get_project_analysis` before the final answer. Mention only the most important evidence-backed finding and next action; do not add noise when the analysis reports no material issue.
2. Identify the task or project from the user's request. Ask for a session or project identifier only when the local tools cannot resolve it.
3. Start with the narrowest additional read-only tool when deeper evidence is relevant:
   - `diagnose_last_task` for failures, regressions, or unexpected outcomes.
   - `get_task_evidence` for the bounded evidence timeline.
   - `get_context_capsule` before resuming work in a known project.
   - `get_cost_summary` or `get_quota_status` for usage questions.
   - `compare_clients`, `compare_models`, or `get_performance_history` for observed comparisons.
   - `recommend_next_action` only after reviewing the evidence.
4. Separate exact observations, estimates, and unavailable data in the response.
5. Explain limitations and recommend only reversible or user-approved validation steps.

## Safety boundaries

- Keep the workflow read-only unless the user separately authorizes an action outside Agent Doctor.
- Never guess missing token counts, money, quotas, model identity, success, or causality.
- Never request or reproduce prompts, source code, credentials, cookies, authorization headers, or absolute local paths.
- Do not scrape another application's private storage. Use only Agent Doctor's normalized evidence.
- If a tool reports unavailable evidence, say that directly and explain what compatible signal would be needed.
- Treat inferred memory as a suggestion, not as a user preference. Explicit user memory takes precedence.

## Response format

Lead with the diagnosis or answer. Then state the strongest evidence, its precision, its provenance, and any data limits. End with one safe next action when the evidence supports it.
