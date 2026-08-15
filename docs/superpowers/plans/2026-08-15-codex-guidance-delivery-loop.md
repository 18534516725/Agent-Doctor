# Codex Guidance Delivery Loop Repair Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Codex guidance operate as an observable pull-based loop: analyze the active task, resolve the current project, record every MCP delivery, and show only controls Codex can actually honor.

**Architecture:** Keep the existing local SQLite evidence and deterministic guidance engine. Extend conversation projection so completed tool messages from an in-progress Codex request become evidence immediately, resolve project-scoped guidance through that project's newest request, persist a bounded delivery receipt when the MCP tool returns guidance, and expose the receipt to the dashboard. The dashboard will distinguish “AI has read the guidance” from “an intervention was triggered” and will limit Codex to observe/guide because MCP cannot enforce tool blocking.

**Tech Stack:** Go 1.25, modernc SQLite, MCP JSON-RPC over stdio, React 19, TypeScript, Vitest, Vite.

---

## File responsibility map

- `internal/guidance/conversation_projector.go`: project evidence already present in an active Codex request without inventing request completion.
- `internal/storage/guidance.go`: resolve newest sessions by request/message activity, evaluate project guidance, and persist/read delivery receipts.
- `migrations/008_guidance_delivery_receipts.sql`: bounded per-session MCP delivery counters and latest delivery facts.
- `internal/app/guidance.go`: return guidance and atomically record that Codex received it.
- `internal/server/routes.go`: include the latest delivery receipt in the active-guidance API.
- `dashboard/src/components/TaskGuardian.tsx`: show delivered/pending/read states and capability-correct controls.
- `plugins/agent-doctor/skills/agent-doctor/SKILL.md` and installer assets: require the current project ID on every coding turn and state visibly when non-continue guidance changes behavior.

### Task 1: Analyze active Codex evidence

**Files:**
- Modify: `internal/guidance/conversation_projector.go`
- Test: `internal/guidance/conversation_projector_test.go`

- [ ] **Step 1: Write a failing active-request test**

Add a test with `StatusCode: 0`, `CompletedAt: nil`, and one tool message. Assert that one `tool.completed` event is projected and no `command.completed` event exists.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/guidance -run TestProjectConversationProjectsAvailableToolEvidenceBeforeRequestCompletion -count=1`

Expected: FAIL because the current completion guard returns no events.

- [ ] **Step 3: Implement the minimal projection change**

Remove the request-level early return. Project tool evidence already captured in `record.Messages`; emit request completion only when `CompletedAt != nil` and the status is successful. Preserve stable IDs and content-free fingerprints.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/guidance -count=1`

Expected: PASS.

### Task 2: Resolve guidance for the active project and record delivery

**Files:**
- Create: `migrations/008_guidance_delivery_receipts.sql`
- Modify: `migrations/embed.go`
- Modify: `internal/guidance/types.go`
- Modify: `internal/storage/guidance.go`
- Modify: `internal/storage/guidance_test.go`
- Modify: `internal/app/guidance.go`
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing storage tests**

Add tests proving that `LatestRuntimeGuidance(ctx, projectID, now)` evaluates the newest active session in that project instead of returning a synthetic quiet decision, and that `RecordGuidanceDelivery` upserts one receipt while incrementing `deliveryCount`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/storage -run 'TestLatestRuntimeGuidanceEvaluatesNewestProjectSession|TestGuidanceDeliveryReceiptTracksReads' -count=1`

Expected: FAIL because project session resolution and delivery storage do not exist.

- [ ] **Step 3: Add the receipt schema and storage API**

Create `guidance_delivery_receipts` keyed by `session_id`, containing project, client, decision kind/id, control level, delivery count, and latest delivery time. Do not store prompts, source, commands, tool results, or credentials. Resolve the latest session using model-request/message activity, scoped by project when supplied.

- [ ] **Step 4: Make MCP delivery observable**

After `runtimeGuidanceEvidence` resolves the decision and control level, record a receipt with client `codex-mcp`. Return a bounded `Delivery receipt` item so the AI and dashboard can correlate the read. A receipt failure must fail the guidance call instead of pretending delivery was recorded.

- [ ] **Step 5: Verify GREEN**

Run: `go test ./internal/storage ./internal/app -count=1`

Expected: PASS.

### Task 3: Expose an honest dashboard state

**Files:**
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/components/TaskGuardian.tsx`
- Modify: `dashboard/tests/taskGuardian.test.tsx`

- [ ] **Step 1: Write failing API and component tests**

Assert `/api/v1/guidance/active` includes `delivery`, and the Codex component renders “AI 已读取本轮指导” plus the delivery time/count. Assert the Codex selector contains only “只观察” and “自动指导”, while a hook-capable Claude status can still expose guard modes.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/server -run TestActiveGuidanceIncludesLatestDelivery -count=1 && npm --prefix dashboard test -- --run taskGuardian`

Expected: FAIL because delivery is absent and all four controls are rendered.

- [ ] **Step 3: Implement delivery-aware UI**

Render three separate facts: monitoring freshness, whether the AI fetched guidance, and whether a non-continue intervention exists. For Codex, label the capability as MCP pull-based advice and hide unsupported enforcement modes. If a saved Codex level is guard/autopilot, show its effective mode as guide with an explicit compatibility note.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/server -count=1 && npm --prefix dashboard test -- --run taskGuardian`

Expected: PASS.

### Task 4: Require project-correct reads and package the repair

**Files:**
- Modify: `plugins/agent-doctor/skills/agent-doctor/SKILL.md`
- Modify: `adapters/claude-code/skills/agent-doctor/SKILL.md`
- Modify: `internal/installer/all.go`
- Modify: related contract tests
- Modify: embedded dashboard assets through the existing build command

- [ ] **Step 1: Write failing integration contract tests**

Require the installed Codex guidance to say: call with the current project ID at the start of every coding turn; re-read after failure/compaction and before completion; follow non-continue instructions; briefly disclose a real intervention to the user; never claim guard enforcement through MCP.

- [ ] **Step 2: Verify RED**

Run: `go test ./plugins/agent-doctor ./internal/installer -count=1`

Expected: FAIL on the new workflow phrases.

- [ ] **Step 3: Update Skill, MCP instructions, and installer assets**

Keep `continue` silent in AI output, but require project-scoped reads and visible disclosure for `advise`, `redirect`, `verify`, or `block`. Keep the limitation that Codex advice is cooperative, not an unenforceable promise.

- [ ] **Step 4: Build and verify the complete release**

Run: `gofmt -w <changed-go-files> && go test ./... -count=1 && npm --prefix dashboard test -- --run && npm --prefix dashboard run build && go generate ./internal/server/web && go test ./... -count=1 && go build ./cmd/agent-doctor`

Expected: all commands PASS and the embedded dashboard contains the delivery-aware UI.

- [ ] **Step 5: Run deterministic installed-flow smoke tests**

Install from the local repository using the supported installer, start the dashboard, invoke `get_runtime_guidance` with the current project ID, and query `/api/v1/guidance/active`. Confirm the response contains a recent delivery receipt and the browser shows the AI-read state.
