# Usable Product and One-Command Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn captured Codex/Claude activity into real runtime guidance, replace empty dashboard shells with actionable local tools, and provide one command that builds, installs, integrates, and starts Agent Doctor.

**Architecture:** Add an idempotent, content-free projection layer between stored conversations and the existing event/guidance engine. Expose focused aggregate repositories for costs, trends, memories, comparisons, integrations, and local data through authenticated loopback APIs. Extend the managed installer to own reversible Codex/Claude blocks and assets, then wrap the complete local workflow in `scripts/install-local.sh`.

**Tech Stack:** Go 1.25, SQLite migrations, React 19, TypeScript, Vite/Vitest, POSIX shell, existing MCP/Hook/Skill assets.

---

### Task 1: Project Codex conversations into runtime guidance events

**Files:**
- Create: `internal/guidance/conversation_projector.go`
- Create: `internal/guidance/conversation_projector_test.go`
- Modify: `internal/storage/conversations.go`
- Modify: `internal/storage/storage_test.go`

- [ ] **Step 1: Write failing projector tests**

Cover these exact behaviors with real `conversations.Request` values:

```go
func TestProjectConversationEmitsContentFreeFailureEvidence(t *testing.T) {
    record := conversations.Request{
        ID: "request-1", SessionID: "session-1", ProjectID: "project-1",
        Client: events.ClientRef{Name: "codex", Version: "1"},
        Model: events.ModelRef{DisplayName: "gpt-public"}, StatusCode: 500,
        StartedAt: time.Unix(1, 0), Messages: []conversations.Message{{
            ID: "tool-1", Role: "tool", ToolName: "exec",
            Content: "private output", ToolPayloadJSON: `{"command":"private command"}`,
        }},
    }
    projected := ProjectConversation(record)
    requireEventTypes(t, projected, events.EventToolFailed)
    encoded, _ := json.Marshal(projected)
    for _, forbidden := range []string{"private output", "private command", "command"} {
        if bytes.Contains(encoded, []byte(forbidden)) { t.Fatalf("leaked %q", forbidden) }
    }
}
```

Also assert stable SHA-256 fingerprints across canonical JSON key order, distinct IDs for distinct messages, request completion/progress classification, and three repeated failures producing `guidance.KindRedirect` after storage round-trip.

- [ ] **Step 2: Run the focused tests and confirm RED**

Run: `go test ./internal/guidance ./internal/storage -run 'ProjectConversation|ConversationGuidance' -count=1`

Expected: FAIL because `ProjectConversation` and storage projection do not exist.

- [ ] **Step 3: Implement the pure projector**

Define:

```go
func ProjectConversation(record conversations.Request) []events.Event
```

Use only request/message IDs, UTC timestamps, bounded client/model/tool labels, status/progress facts, and `sha256:` fingerprints of canonical tool JSON/result content. Map HTTP/provider failure status to `tool.failed`; successful write-like tools to progress; read/search tools to inspection completions; completed requests to command/session progress without copying content.

- [ ] **Step 4: Persist projections in the same conversation transaction**

Extract a transaction-scoped event insert helper in `internal/storage/events.go`. Call it from `SaveConversationRequest` for every projected event with `ON CONFLICT(id) DO NOTHING`. Preserve the conversation when projection returns no compatible signals; transaction errors remain generic.

- [ ] **Step 5: Verify GREEN and privacy regressions**

Run: `go test ./internal/guidance ./internal/storage ./internal/privacy -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/guidance internal/storage
git commit -m "feat: derive runtime guidance from Codex conversations"
```

### Task 2: Make guidance availability honest and localize controls

**Files:**
- Create: `internal/guidance/status.go`
- Create: `internal/guidance/status_test.go`
- Modify: `internal/storage/guidance.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/components/TaskGuardian.tsx`
- Modify: `dashboard/tests/taskGuardian.test.tsx`

- [ ] **Step 1: Write failing backend and React status tests**

Assert no events returns `unavailable`, recent events without confirmed feedback return `observing`, recent compatible guidance returns `active`, stale evidence returns `stale`, and repository failure returns `error`. In React assert zero evidence renders “尚未建立引导链路”, not “正在推进”. Assert Chinese select options are `观察`, `引导`, `守护`, `自动守护`, while values remain `observe|guide|guard|autopilot`.

- [ ] **Step 2: Run focused tests and confirm RED**

Run:

```bash
go test ./internal/guidance ./internal/server -run 'GuidanceStatus|GuidanceAPI' -count=1
pnpm --dir dashboard test --run taskGuardian.test.tsx
```

Expected: FAIL because availability/capability fields and localized labels are absent.

- [ ] **Step 3: Implement status and capability contracts**

Add:

```go
type ConnectionState string
const (
    StateActive ConnectionState = "active"
    StateObserving ConnectionState = "observing"
    StateStale ConnectionState = "stale"
    StateUnavailable ConnectionState = "unavailable"
    StateError ConnectionState = "error"
)
type Status struct {
    State ConnectionState `json:"state"`
    Client string `json:"client"`
    Advice bool `json:"advice"`
    Enforcement bool `json:"enforcement"`
    LastEvidenceAt *time.Time `json:"lastEvidenceAt,omitempty"`
    Explanation string `json:"explanation"`
}
```

Return status alongside `/api/v1/guidance/active`. Codex capability is advice-only; Claude Hook is advice plus limited enforcement.

- [ ] **Step 4: Implement honest UI states and Chinese labels**

Keep stored enum values stable. Derive headings from status and decision. Add a short capability statement beside the selector. Never render a fake health score or “on track” for unavailable/observing/stale/error.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/guidance ./internal/server -count=1 && pnpm --dir dashboard test --run`

```bash
git add internal/guidance internal/storage internal/server dashboard
git commit -m "fix: show honest guidance connection states"
```

### Task 3: Replace the token billboard with cost intelligence and real trends

**Files:**
- Create: `internal/insights/types.go`
- Create: `internal/storage/insights.go`
- Create: `internal/storage/insights_test.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/App.tsx`
- Rewrite: `dashboard/src/pages/CostsPage.tsx`
- Rewrite: `dashboard/src/pages/TrendsPage.tsx`
- Create: `dashboard/tests/costsAndTrends.test.tsx`
- Modify: `dashboard/src/styles.css`

- [ ] **Step 1: Write failing aggregate tests**

Insert requests with input, cached, output, reasoning, exact/estimated/unavailable cost, status, model, client and dates. Assert:

```go
if got.Usage.UncachedInputTokens != got.Usage.InputTokens-got.Usage.CachedTokens { t.Fatal(...) }
if got.Cost.UnknownRequests != 1 || got.Cost.Availability != "partial" { t.Fatal(...) }
if got.Trends[0].P95LatencyMS < got.Trends[0].P50LatencyMS { t.Fatal(...) }
```

Require rankings by project/session/client/model, bounded unknown request records, and 7/30-day buckets sourced from `model_requests`, not `events`.

- [ ] **Step 2: Write failing UI tests**

Assert the first cost heading is “费用暂不可计算” when all amounts are unavailable, displays `未缓存输入 82.9 万`, `缓存率 95.3%`, and provides an unknown-price explanation. Assert trends render request count, failure rate, P50/P95 latency, uncached tokens and cache rate from request buckets.

- [ ] **Step 3: Run tests and confirm RED**

Run: `go test ./internal/storage ./internal/server -run 'CostIntelligence|RequestTrends' -count=1 && pnpm --dir dashboard test --run costsAndTrends.test.tsx`

- [ ] **Step 4: Implement aggregate repository and APIs**

Add `GET /api/v1/insights/costs?days=30` and `GET /api/v1/insights/trends?days=30`. Clamp days to 7 or 30, rank at most 20 rows, keep exact/estimated/unknown separate, calculate percentiles deterministically in Go, and return limitations for missing model price mappings.

- [ ] **Step 5: Implement action-first pages**

Costs order: amount availability → unknown repair action → usage composition → rankings → bounded unknown records. Trends order: range selector → request/failure summary → latency P50/P95 → token/cache chart → sample limitation. Remove the giant total-token hero.

- [ ] **Step 6: Verify and commit**

Run: `go test ./internal/storage ./internal/server -count=1 && pnpm --dir dashboard test --run && pnpm --dir dashboard build`

```bash
git add internal/insights internal/storage internal/server dashboard
git commit -m "feat: add actionable cost and request trend insights"
```

### Task 4: Build a user-confirmed project memory workflow

**Files:**
- Create: `internal/memory/types.go`
- Create: `internal/storage/memory.go`
- Create: `internal/storage/memory_test.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Rewrite: `dashboard/src/pages/MemoryPage.tsx`
- Modify: `dashboard/src/components/ConversationTimeline.tsx`
- Create: `dashboard/tests/memory.test.tsx`

- [ ] **Step 1: Write failing storage/API tests**

Test create manual candidate, save a selected conversation message with explicit user action, confirm candidate to active, edit content, disable, delete, list by project/status, and reject content over 16 KiB or unknown fields. Every item must retain source kind/source ID and timestamps.

- [ ] **Step 2: Write failing React tests**

Assert MemoryPage lists real rows, filters candidate/active/disabled, confirms a candidate, edits a memory, disables it, and ConversationTimeline calls “保存为项目记忆” only after the user clicks the message action.

- [ ] **Step 3: Run RED tests**

Run: `go test ./internal/storage ./internal/server -run Memory -count=1 && pnpm --dir dashboard test --run memory.test.tsx`

- [ ] **Step 4: Implement memory repository and routes**

Use existing `memories` columns with content stored only after explicit user action. Add authenticated GET/POST/PATCH/DELETE routes under `/api/v1/projects/{id}/memories`. Enforce state transitions candidate→active, active↔disabled, and deletion. Publish `memory.changed` SSE.

- [ ] **Step 5: Implement the real memory workspace**

Provide add form, status tabs, provenance, edit/confirm/disable/delete actions and empty-state instructions. Do not auto-copy prompts. Update context capsule lookup to include only active memories with provenance.

- [ ] **Step 6: Verify and commit**

Run: `go test ./internal/storage ./internal/server ./internal/context -count=1 && pnpm --dir dashboard test --run`

```bash
git add internal/memory internal/storage internal/server internal/context dashboard
git commit -m "feat: add confirmed project memory workflow"
```

### Task 5: Make task comparison useful from two sessions onward

**Files:**
- Create: `internal/storage/comparisons.go`
- Create: `internal/storage/comparisons_test.go`
- Modify: `internal/comparison/compare.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Rewrite: `dashboard/src/pages/ComparisonPage.tsx`
- Create: `dashboard/tests/comparison.test.tsx`

- [ ] **Step 1: Write failing comparison tests**

Create two sessions in one project. Assert immediate descriptive comparison of request count, failures, P50/P95 latency, uncached/output tokens, cache rate and cost availability. Assert reversed IDs normalize to one stable comparison ID. Assert fewer than 15 matched samples returns `winner: null` and `samplesNeeded`, while 15+ enables only an observational cohort result.

- [ ] **Step 2: Run RED tests**

Run: `go test ./internal/comparison ./internal/storage ./internal/server -run Comparison -count=1 && pnpm --dir dashboard test --run comparison.test.tsx`

- [ ] **Step 3: Implement session summaries and idempotent comparison storage**

Add session selector/list API and `POST /api/v1/comparisons`. Save result JSON with algorithm version and sorted input IDs into existing `comparisons`. Never infer task semantics from prompt text.

- [ ] **Step 4: Implement the comparison workspace**

Show two selectors, metric rows with absolute and percentage differences, evidence precision, and an explicit cohort gate “还差 N 个匹配样本”. Remove the single empty count hero.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/comparison ./internal/storage ./internal/server -count=1 && pnpm --dir dashboard test --run`

```bash
git add internal/comparison internal/storage internal/server dashboard
git commit -m "feat: compare real captured sessions"
```

### Task 6: Turn integrations and local data into operational tools

**Files:**
- Create: `internal/localdata/types.go`
- Create: `internal/storage/localdata.go`
- Create: `internal/storage/localdata_test.go`
- Modify: `internal/installer/detect.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Rewrite: `dashboard/src/pages/IntegrationsPage.tsx`
- Rewrite: `dashboard/src/pages/PrivacyPage.tsx`
- Create: `dashboard/tests/operations.test.tsx`

- [ ] **Step 1: Write failing local inventory and capability tests**

Assert database byte size, table counts, earliest/latest timestamps, retention days and expired-record estimate. Assert each integration returns installed/configured/active, last evidence, observation/advice/enforcement booleans, restart requirement and repair action availability.

- [ ] **Step 2: Run RED tests**

Run: `go test ./internal/storage ./internal/server -run 'LocalData|IntegrationCapabilities' -count=1 && pnpm --dir dashboard test --run operations.test.tsx`

- [ ] **Step 3: Implement read APIs and controlled actions**

Add `GET /api/v1/local-data`, `POST /api/v1/local-data/cleanup`, and richer `GET /api/v1/connections`. Cleanup requires `{before, confirm:true}`, uses a transaction, returns deleted counts, and publishes SSE. Keep existing per-session delete and privacy settings.

- [ ] **Step 4: Implement operational pages**

Connections become a capability matrix with last activity, missing step and restart note. Local Data shows inventory, retention, cleanup preview, export entry and destructive actions with two-stage confirmation.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/storage ./internal/server ./internal/installer -count=1 && pnpm --dir dashboard test --run`

```bash
git add internal/localdata internal/storage internal/installer internal/server dashboard
git commit -m "feat: add integration and local data operations"
```

### Task 7: Install every supported local integration with one command

**Files:**
- Modify: `internal/installer/managed.go`
- Modify: `internal/installer/installer_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Create: `scripts/install-local.sh`
- Create: `scripts/install-local.test.sh`
- Modify: `install.sh`
- Modify: `docs/install.md`
- Modify: `docs/usage.md`

- [ ] **Step 1: Write failing managed integration tests**

In a temporary HOME assert `BuildAllIntegrationsPlan` creates exactly these managed/reversible assets without duplicates:

```text
~/.codex/config.toml                         Agent Doctor MCP block
~/.codex/skills/agent-doctor/SKILL.md       Codex Skill
~/.codex/AGENTS.md                          Agent Doctor guidance block
~/.claude/hooks/agent-doctor.json           Claude Hook bundle
~/.claude/skills/agent-doctor/SKILL.md       Claude Skill
```

Reapply twice and assert byte-identical output. Uninstall removes only owned blocks/files and restores user content.

- [ ] **Step 2: Write failing shell installation test**

`scripts/install-local.test.sh` must use temporary HOME/PATH and stubbed `pnpm`, `go`, and browser/start calls to assert the script checks dependencies, builds in order, atomically installs under `~/.local/bin`, invokes `setup --yes --all --json`, and starts exactly once. Also assert a missing dependency exits nonzero with its install command.

- [ ] **Step 3: Run RED tests**

Run: `go test ./internal/installer ./internal/app -run 'AllIntegrations|SetupAll' -count=1 && sh scripts/install-local.test.sh`

- [ ] **Step 4: Implement `setup --all` and reversible assets**

Embed shipped Skill/Hook files in the Go binary with `//go:embed`. Extend setup parsing to accept `--all`; keep current Codex-only behavior without it. Build one installer plan so Apply provides a single rollback boundary. Return JSON listing applied assets and clients requiring restart.

- [ ] **Step 5: Implement the local one-command script**

Use POSIX `sh`, explicit paths, `mktemp -d`, trap cleanup, and no `eval`. Run dependency checks, `pnpm install --frozen-lockfile`, `./scripts/embed-dashboard.sh`, focused install tests, `go build`, install to a temporary sibling then `mv`, setup all, and finally `exec "$HOME/.local/bin/agent-doctor" start`.

- [ ] **Step 6: Update verified release installer**

After checksum verification and binary install, call `setup --yes --all --json` and `exec agent-doctor start`. Preserve environment override to skip auto-start for CI: `AGENT_DOCTOR_NO_START=1`.

- [ ] **Step 7: Verify and commit**

Run: `go test ./internal/installer ./internal/app -count=1 && sh scripts/install-local.test.sh && shellcheck scripts/install-local.sh install.sh` when shellcheck is available.

```bash
git add internal/installer internal/app scripts/install-local.sh scripts/install-local.test.sh install.sh docs/install.md docs/usage.md
git commit -m "feat: add one-command local installation"
```

### Task 8: Embed, dogfood, and release-verify the complete workspace

**Files:**
- Modify: `README.md`
- Modify: `docs/compatibility.md`
- Modify: `docs/privacy.md`
- Modify: `internal/server/web/index.html`
- Replace: `internal/server/web/assets/*`

- [ ] **Step 1: Update user-facing documentation**

Put `./scripts/install-local.sh` as the first local-clone command. Document that it opens and occupies the terminal, clients need restart, Codex guidance is advisory, Claude Hook enforcement is bounded, and unknown costs/missing signals remain unavailable.

- [ ] **Step 2: Rebuild embedded dashboard**

Run: `./scripts/embed-dashboard.sh`

Expected: one HTML, one CSS and one JS embedded asset set with current hashes.

- [ ] **Step 3: Run full verification**

Run:

```bash
go test ./... -race -count=1
go vet ./...
pnpm --dir dashboard test --run
pnpm --dir dashboard build
sh scripts/install-local.test.sh
./scripts/check-secrets.sh
./scripts/check-docs.sh
git diff --check
```

Expected: every command exits 0.

- [ ] **Step 4: Run real temporary-HOME smoke test**

Build a binary into `mktemp -d`, run `setup --yes --all --json` with a temporary HOME, verify five managed assets, start the loopback server with `--once`, save three identical Codex failure records, and verify active guidance returns redirect without raw payload text.

- [ ] **Step 5: Commit release-ready artifacts**

```bash
git add README.md docs/compatibility.md docs/privacy.md internal/server/web
git commit -m "docs: ship the usable Agent Doctor workflow"
```
