# Cross-Client Task Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Codex and Claude Code resume the same local project with a bounded, provenance-labelled handoff containing confirmed project memory and the latest captured task state.

**Architecture:** Add a client-neutral `handoff` domain model generated deterministically from SQLite. Codex receives it through `get_context_capsule`; Claude Code receives the same rendered capsule through its documented `SessionStart` hook. Delivery receipts and a dashboard preview make the transfer visible without copying complete transcripts by default.

**Tech Stack:** Go 1.25, SQLite migrations, MCP JSON-RPC, Claude Code hooks, React 19, TypeScript, Vitest.

---

### Task 1: Define the handoff domain and deterministic renderer

**Files:**
- Create: `internal/handoff/types.go`
- Create: `internal/handoff/render.go`
- Test: `internal/handoff/render_test.go`

- [ ] Write failing tests proving the renderer includes a goal, latest result, confirmed memories, source client/session, provenance and limitations while respecting an 800-token budget and filtering credentials.
- [ ] Run `go test ./internal/handoff -run Test -count=1` and confirm the new tests fail because the package is absent.
- [ ] Implement the minimum domain types and deterministic renderer.
- [ ] Re-run the focused tests and confirm they pass.

### Task 2: Build and persist project handoffs from SQLite

**Files:**
- Create: `migrations/009_handoff_delivery_receipts.sql`
- Create: `internal/storage/handoff.go`
- Test: `internal/storage/handoff_test.go`

- [ ] Write failing storage tests proving only `active` memories are selected, the newest captured request supplies bounded user/assistant context, raw and hashed working-directory identities can resolve the same project, and a delivery receipt is recorded.
- [ ] Run the focused storage tests and verify the expected missing-method/schema failure.
- [ ] Implement `ProjectHandoff`, identity candidate lookup, capsule persistence and delivery receipt queries.
- [ ] Re-run the focused storage tests and migration tests.

### Task 3: Connect MCP and Claude Code to the same capsule

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `internal/installer/all.go`
- Modify: `internal/installer/installer_test.go`
- Modify: `plugins/agent-doctor/skills/agent-doctor/SKILL.md`
- Modify: `adapters/claude-code/skills/agent-doctor/SKILL.md`

- [ ] Write failing tests proving `get_context_capsule` returns real local handoff evidence and records a Codex delivery.
- [ ] Write a failing hook test proving `SessionStart` injects the same capsule even when runtime guidance is `continue`, and records a Claude Code delivery.
- [ ] Update installed Codex and Claude skills so project resume fetches shared context explicitly.
- [ ] Implement the MCP and hook paths, keeping hook failure fail-open and avoiding prompt rewriting or permission escalation.
- [ ] Run focused app and installer tests.

### Task 4: Expose transfer visibility in the local dashboard

**Files:**
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/pages/MemoryPage.tsx`
- Modify: `dashboard/src/styles.css`
- Test: `dashboard/tests/memoryPage.test.tsx`

- [ ] Write failing HTTP and component tests for a handoff preview containing source client, task summary, confirmed-memory count, last delivery target/time and explicit limitations.
- [ ] Add `GET /api/v1/projects/{id}/handoff` and typed dashboard API support.
- [ ] Add a Chinese-first “跨 AI 任务接力” panel to Project Memory with transparent transfer contents and empty/error states.
- [ ] Run server and dashboard tests.

### Task 5: Verify, install, and commit on main

**Files:**
- Modify generated embedded dashboard assets through the existing build workflow.
- Modify: `docs/usage.md`
- Modify: `adapters/claude-code/README.md`

- [ ] Run `gofmt` on changed Go files.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `npm test -- --run` and `npm run build` in `dashboard`.
- [ ] Refresh embedded dashboard assets with the repository build workflow and re-run Go tests.
- [ ] Run the one-command setup/install path and a real MCP plus synthetic Claude `SessionStart` smoke test against an isolated temporary data directory.
- [ ] Inspect `git diff --check` and `git status`, preserving unrelated `.superpowers/` files.
- [ ] Commit the verified implementation directly on `main`.
