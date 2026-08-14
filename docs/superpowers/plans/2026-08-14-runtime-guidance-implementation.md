# Runtime Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working Agent Doctor reliability loop that turns sanitized runtime evidence into deduplicated guidance, exposes it to coding agents, feeds it back through Claude Code hooks, and replaces the dashboard’s metric-first overview with an active-task guardian.

**Architecture:** Add a pure `internal/guidance` package that projects normalized events and evaluates four conservative rules without database or UI dependencies. Persist emitted decisions and project control levels in SQLite, expose the latest decision through MCP and the loopback API, and let the Claude Code adapter translate decisions into documented hook responses while remaining fail-open on internal errors. The React dashboard consumes the same decision model and keeps raw transcripts and usage as expandable evidence.

**Tech Stack:** Go 1.25, modernc SQLite, JSON-RPC MCP over stdio, Claude Code command hooks, React 19, TypeScript 7, Vitest, Vite.

---

## File responsibility map

- `internal/guidance/types.go`: stable guidance enums, decision payload, projected signal, and control level.
- `internal/guidance/project.go`: convert sanitized `events.Event` values into bounded signals; never expose raw payloads.
- `internal/guidance/engine.go`: deterministic rule priority, decision IDs, evidence fingerprints, expiration, and deduplication inputs.
- `internal/guidance/engine_test.go`: table-driven evidence/rule coverage.
- `migrations/006_runtime_guidance.sql`: persisted decisions and project control levels.
- `internal/storage/guidance.go`: latest-session resolution, projection input loading, idempotent decision persistence, list/read settings.
- `internal/adapters/claudecode/normalize.go`: store non-reversible input/result fingerprints and serialize bounded hook guidance.
- `internal/app/guidance.go`: shared runtime guidance service used by hooks, MCP, and HTTP.
- `internal/app/app.go`: wire runtime guidance into the existing Claude hook and MCP entrypoints.
- `internal/mcp/tools.go`: declare `get_runtime_guidance`.
- `internal/server/routes.go`: expose active guidance and project control-level endpoints.
- `dashboard/src/components/TaskGuardian.tsx`: action-first active task and latest intervention UI.
- `dashboard/src/api.ts`, `dashboard/src/App.tsx`, `dashboard/src/pages/types.ts`: load and distribute guidance state.
- `dashboard/src/pages/OverviewPage.tsx`, `dashboard/src/styles.css`: make task guidance the primary overview content.
- `plugins/agent-doctor/skills/agent-doctor/SKILL.md`, `adapters/claude-code/skills/agent-doctor/SKILL.md`: require runtime guidance at start, after repeated failures, after compaction, and before completion.

### Task 1: Pure deterministic guidance engine

**Files:**
- Create: `internal/guidance/types.go`
- Create: `internal/guidance/project.go`
- Create: `internal/guidance/engine.go`
- Test: `internal/guidance/engine_test.go`

- [ ] **Step 1: Write failing rule tests**

Create tests that express the four supported rules and the quiet healthy state:

```go
func TestEvaluateRepeatedFailureRedirectsAfterThreeIdenticalFailures(t *testing.T) {
    now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
    signals := []Signal{
        {EventID: "1", Kind: SignalToolFailed, Tool: "Bash", InputFingerprint: "same", ResultFingerprint: "error", At: now.Add(-3 * time.Minute)},
        {EventID: "2", Kind: SignalToolFailed, Tool: "Bash", InputFingerprint: "same", ResultFingerprint: "error", At: now.Add(-2 * time.Minute)},
        {EventID: "3", Kind: SignalToolFailed, Tool: "Bash", InputFingerprint: "same", ResultFingerprint: "error", At: now.Add(-time.Minute)},
    }
    got := Evaluate(SessionState{SessionID: "session-1", ProjectID: "project-1", Signals: signals}, now)
    if got.Kind != KindRedirect || got.Severity != SeverityHigh || len(got.Evidence) != 3 {
        t.Fatalf("decision=%+v", got)
    }
}

func TestEvaluateRepeatedReadWithoutProgressAdvises(t *testing.T) {
    now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
    signals := make([]Signal, 4)
    for index := range signals {
        signals[index] = Signal{EventID: fmt.Sprintf("e-%d", index), Kind: SignalToolCompleted, Tool: "Read", InputFingerprint: "same-file", At: now.Add(time.Duration(index) * time.Second)}
    }
    got := Evaluate(SessionState{SessionID: "session-1", ProjectID: "project-1", Signals: signals}, now)
    if got.Kind != KindAdvise || got.Finding != "Repeated activity produced no observable progress." {
        t.Fatalf("decision=%+v", got)
    }
}

func TestEvaluateCompactionReturnsHandoffGuidance(t *testing.T) {
    now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
    got := Evaluate(SessionState{SessionID: "s", ProjectID: "p", Signals: []Signal{{EventID: "compact", Kind: SignalContextCompacted, At: now}}}, now)
    if got.Kind != KindAdvise || !strings.Contains(got.Instruction, "preserve") {
        t.Fatalf("decision=%+v", got)
    }
}

func TestEvaluateStopWithoutSuccessfulValidationRequestsVerification(t *testing.T) {
    now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
    state := SessionState{SessionID: "s", ProjectID: "p", Signals: []Signal{{EventID: "stop", Kind: SignalSessionCompleted, At: now}}}
    if got := Evaluate(state, now); got.Kind != KindVerify || got.Severity != SeverityHigh {
        t.Fatalf("decision=%+v", got)
    }
    state.Signals = append([]Signal{{EventID: "validation", Kind: SignalValidationPassed, At: now.Add(-time.Second)}}, state.Signals...)
    if got := Evaluate(state, now); got.Kind != KindContinue {
        t.Fatalf("validated decision=%+v", got)
    }
}

func TestEvaluateHealthySessionIsQuiet(t *testing.T) {
    got := Evaluate(SessionState{SessionID: "s", ProjectID: "p"}, time.Now())
    if got.Kind != KindContinue || got.Severity != SeverityInfo || got.Instruction != "" {
        t.Fatalf("decision=%+v", got)
    }
}
```

- [ ] **Step 2: Run the package test and confirm the missing-package failure**

Run: `go test ./internal/guidance -count=1`

Expected: FAIL because `Signal`, `SessionState`, and `Evaluate` do not exist.

- [ ] **Step 3: Implement the stable domain types**

Define exact JSON-facing types in `types.go`:

```go
type Kind string
const (
    KindContinue Kind = "continue"
    KindAdvise Kind = "advise"
    KindRedirect Kind = "redirect"
    KindAsk Kind = "ask"
    KindBlock Kind = "block"
    KindVerify Kind = "verify"
)

type Severity string
const (
    SeverityInfo Severity = "info"
    SeverityWarning Severity = "warning"
    SeverityHigh Severity = "high"
    SeverityCritical Severity = "critical"
)

type ControlLevel string
const (
    ControlObserve ControlLevel = "observe"
    ControlGuide ControlLevel = "guide"
    ControlGuard ControlLevel = "guard"
    ControlAutopilot ControlLevel = "autopilot"
)

type Decision struct {
    DecisionID string `json:"decisionId"`
    SessionID string `json:"sessionId"`
    ProjectID string `json:"projectId"`
    Kind Kind `json:"kind"`
    Severity Severity `json:"severity"`
    Finding string `json:"finding"`
    Evidence []string `json:"evidence"`
    Confidence string `json:"confidence"`
    Instruction string `json:"instruction"`
    ProhibitedActions []string `json:"prohibitedActions"`
    Verification []string `json:"verification"`
    EvidenceFingerprint string `json:"evidenceFingerprint"`
    ExpiresAt time.Time `json:"expiresAt"`
    CreatedAt time.Time `json:"createdAt"`
}
```

Add `SignalKind`, `Signal`, and `SessionState`. Keep `Signal` limited to event IDs, timestamps, bounded public tool labels, non-reversible fingerprints, and booleans such as `Progress`; it must not contain raw commands, paths, prompts, source, or tool output.

- [ ] **Step 4: Implement projection and rule priority**

Implement `Project(events []events.Event) SessionState` and `Evaluate(state SessionState, now time.Time) Decision` with this priority:

1. stop without a newer successful validation → `verify`;
2. three matching trailing failures → `redirect`;
3. most recent event is compaction → `advise` with handoff instruction;
4. four matching read/search completions without a progress signal → `advise`;
5. otherwise → quiet `continue`.

Create `DecisionID` as `sha256(sessionID + "\x00" + kind + "\x00" + evidenceFingerprint)`, use only source event IDs in `Evidence`, and set non-continue expiry to `now.Add(10*time.Minute)`.

- [ ] **Step 5: Run the engine tests**

Run: `go test ./internal/guidance -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the pure engine**

```bash
git add internal/guidance
git commit -m "feat: add deterministic runtime guidance engine"
```

### Task 2: Privacy-safe Claude evidence fingerprints

**Files:**
- Modify: `internal/adapters/claudecode/normalize.go`
- Test: `internal/adapters/claudecode/normalize_test.go`

- [ ] **Step 1: Add failing privacy and stability tests**

Extend the existing official-hook test to assert `toolInputFingerprint` and `toolResultFingerprint` exist, contain `sha256:`, and do not contain the command, secret, path, or output. Add a second normalization of the same semantic JSON with different whitespace and assert the input fingerprints match.

- [ ] **Step 2: Run the adapter test and confirm failure**

Run: `go test ./internal/adapters/claudecode -run 'TestNormalizeOfficialHookAllowlistsEvidence|TestToolFingerprintsAreStable' -count=1`

Expected: FAIL because fingerprints are not yet emitted.

- [ ] **Step 3: Parse raw tool fields only long enough to hash them**

Add bounded `json.RawMessage` fields for `tool_input` and `tool_response`. Canonicalize valid JSON by unmarshalling into `any` and re-marshalling before SHA-256. Hash invalid JSON bytes directly. Emit only:

```json
{
  "toolName": "Bash",
  "toolInputFingerprint": "sha256:...",
  "toolResultFingerprint": "sha256:..."
}
```

Do not store lengths, prefixes, arguments, paths, keys, values, response categories, or error text. Preserve the existing 256 KiB envelope limit.

- [ ] **Step 4: Run adapter and privacy suites**

Run: `go test ./internal/adapters/claudecode ./internal/privacy -count=1`

Expected: PASS and no secret appears in failure output.

- [ ] **Step 5: Commit fingerprint capture**

```bash
git add internal/adapters/claudecode
git commit -m "feat: fingerprint Claude tool evidence safely"
```

### Task 3: Persist guidance and control levels

**Files:**
- Create: `migrations/006_runtime_guidance.sql`
- Create: `internal/storage/guidance.go`
- Modify: `internal/storage/storage_test.go`
- Test: `internal/storage/guidance_test.go`

- [ ] **Step 1: Write migration and repository tests first**

Assert schema version 6 and tables `guidance_decisions` and `project_guidance_settings`. Add a round-trip test that inserts three identical failure events, calls `RuntimeGuidance(ctx, sessionID, now)` twice, receives the same decision ID, and finds one persisted row. Add settings tests for default `guide`, saving `guard`, and rejecting an unsupported level.

- [ ] **Step 2: Run storage tests and confirm schema failure**

Run: `go test ./internal/storage -run 'TestOpenMigrates|TestRuntimeGuidance|TestGuidanceControlLevel' -count=1`

Expected: FAIL with schema version/table/method errors.

- [ ] **Step 3: Add the migration**

Create two tables and indexes:

```sql
CREATE TABLE guidance_decisions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('continue','advise','redirect','ask','block','verify')),
    severity TEXT NOT NULL CHECK (severity IN ('info','warning','high','critical')),
    payload_json TEXT NOT NULL,
    evidence_fingerprint TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(session_id, evidence_fingerprint)
);

CREATE TABLE project_guidance_settings (
    project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    control_level TEXT NOT NULL DEFAULT 'guide' CHECK (control_level IN ('observe','guide','guard','autopilot')),
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_guidance_session_created ON guidance_decisions(session_id, created_at DESC);
CREATE INDEX idx_guidance_project_created ON guidance_decisions(project_id, created_at DESC);
```

- [ ] **Step 4: Implement storage methods**

Implement:

```go
func (database *DB) RuntimeGuidance(ctx context.Context, sessionID string, now time.Time) (guidance.Decision, error)
func (database *DB) LatestRuntimeGuidance(ctx context.Context, projectID string, now time.Time) (guidance.Decision, error)
func (database *DB) ListActiveGuidance(ctx context.Context, now time.Time, limit int) ([]guidance.Decision, error)
func (database *DB) GuidanceControlLevel(ctx context.Context, projectID string) (guidance.ControlLevel, error)
func (database *DB) SaveGuidanceControlLevel(ctx context.Context, projectID string, level guidance.ControlLevel, now time.Time) error
```

When `sessionID` is empty, resolve the most recently updated session. Project events using `guidance.Project`, evaluate, persist non-continue decisions with `INSERT ... ON CONFLICT DO NOTHING`, and return existing payload for the same fingerprint. Never persist quiet `continue` decisions.

- [ ] **Step 5: Run storage and migration tests**

Run: `go test ./internal/storage ./migrations -count=1`

Expected: PASS with schema version 6.

- [ ] **Step 6: Commit persistence**

```bash
git add migrations/006_runtime_guidance.sql internal/storage
git commit -m "feat: persist runtime guidance decisions"
```

### Task 4: Expose runtime guidance through MCP and update the skills

**Files:**
- Modify: `internal/mcp/tools.go`
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`
- Create: `internal/app/guidance.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `plugins/agent-doctor/skills/agent-doctor/SKILL.md`
- Modify: `adapters/claude-code/skills/agent-doctor/SKILL.md`
- Modify: `adapters/cursor/rules/agent-doctor.mdc`

- [ ] **Step 1: Add a failing MCP contract test**

Add `get_runtime_guidance` after `get_project_analysis`, with optional bounded `sessionId` and `projectId`. Assert MCP initialize instructions mention runtime guidance, and the call returns finding, instruction, evidence IDs, control level, provenance, precision, and limitations without prompts, paths, or raw payloads.

- [ ] **Step 2: Run MCP and app backend tests**

Run: `go test ./internal/mcp ./internal/app -run 'RuntimeGuidance|ToolList|Initialize' -count=1`

Expected: FAIL because the tool and backend branch do not exist.

- [ ] **Step 3: Add a shared application guidance service**

Define a narrow store interface in `internal/app/guidance.go` and a formatter that maps `guidance.Decision` to `mcp.ToolEvidence`. Use:

- summary = finding for active guidance, otherwise “No evidence-backed intervention is required.”;
- items = decision kind, severity, instruction, evidence IDs, prohibited actions, verification;
- provenance = `local-sqlite-deterministic-guidance`;
- precision = `exact` only when all source events are exact, otherwise `estimated` or `unavailable`;
- data limits = explicit client capability and missing evidence statements.

- [ ] **Step 4: Register the tool and backend execution path**

Add the definition, update the fixed-order tool test, extend `localMCPBackend.Execute`, and update initialization instructions so clients query guidance at task start, after repeated failures or compaction, and before final completion.

- [ ] **Step 5: Rewrite shipped skills around the feedback loop**

Both skills must say:

1. call `get_runtime_guidance` at task start;
2. call again after two similar failures, after context compaction, and before claiming completion;
3. follow `redirect`/`verify` guidance when supported by evidence;
4. never claim a hook blocked an action when the client is only MCP/Skill capable;
5. treat `continue` as silence and avoid adding diagnostic noise.

Keep all tools read-only and retain current privacy prohibitions.

- [ ] **Step 6: Run MCP, app, and adapter contracts**

Run: `go test ./internal/mcp ./internal/app ./plugins/agent-doctor ./adapters/cursor -count=1`

Expected: PASS.

- [ ] **Step 7: Commit MCP and skills**

```bash
git add internal/mcp internal/app/guidance.go internal/app/app.go internal/app/app_test.go plugins/agent-doctor adapters/claude-code/skills adapters/cursor/rules
git commit -m "feat: guide coding agents from runtime evidence"
```

### Task 5: Feed guidance back through Claude Code hooks

**Files:**
- Modify: `internal/adapters/claudecode/normalize.go`
- Modify: `internal/adapters/claudecode/normalize_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `adapters/claude-code/hooks/hooks.json`
- Modify: `adapters/claude-code/contract_test.go`

- [ ] **Step 1: Write failing response-mapping tests**

Cover these exact mappings:

- `guide` + `advise|redirect|verify` on SessionStart/PostToolUse/PreCompact → `hookSpecificOutput.additionalContext`;
- `guard|autopilot` + `block` on PreToolUse → documented `permissionDecision: "deny"` with bounded reason;
- `guard|autopilot` + `verify` on Stop → top-level `decision: "block"` and bounded reason;
- `observe`, `continue`, expired decision, storage failure, or timeout → empty stdout and exit 0.

- [ ] **Step 2: Run focused hook tests and confirm failure**

Run: `go test ./internal/adapters/claudecode ./internal/app -run 'Guidance|ClaudeHook' -count=1`

Expected: FAIL because guidance response mapping is absent.

- [ ] **Step 3: Implement bounded hook responses**

Add:

```go
type GuidanceResponse struct {
    Decision string `json:"decision,omitempty"`
    Reason string `json:"reason,omitempty"`
    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

func ResponseForDecision(hookName string, level guidance.ControlLevel, decision guidance.Decision, tokenBudget int) (GuidanceResponse, bool)
```

Limit returned guidance to 800 estimated tokens, remove absolute paths and credential patterns with the existing privacy filter, and never echo event payloads. Do not return a blocking response in `observe` or `guide` mode.

- [ ] **Step 4: Wire ingest → evaluate → respond**

Replace the current write-only Claude hook flow with a function that reads the envelope once, normalizes and inserts the event, queries the new decision, loads project control level, writes JSON only when a response exists, and otherwise remains silent. Any internal error still returns process exit code 0 and a generic stderr diagnostic.

Add `PreToolUse` to `hooks.json`. Keep the current 0.5 second hook timeout and ensure evaluation is local SQLite + pure Go only.

- [ ] **Step 5: Run hook, app, privacy, and contract tests**

Run: `go test ./internal/adapters/claudecode ./internal/app ./internal/privacy ./adapters/claude-code -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the first closed-loop adapter**

```bash
git add internal/adapters/claudecode internal/app adapters/claude-code
git commit -m "feat: return runtime guidance through Claude hooks"
```

### Task 6: Add the loopback guidance API and task guardian UI

**Files:**
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/pages/types.ts`
- Create: `dashboard/src/components/TaskGuardian.tsx`
- Create: `dashboard/tests/taskGuardian.test.tsx`
- Modify: `dashboard/src/pages/OverviewPage.tsx`
- Modify: `dashboard/src/styles.css`
- Modify: `dashboard/tests/pages.test.tsx`

- [ ] **Step 1: Write failing HTTP tests**

Add authenticated tests for:

- `GET /api/v1/guidance/active` returning `{items: []}` when quiet;
- active guidance returning only structured decisions;
- `GET /api/v1/projects/{id}/guidance` defaulting to `guide`;
- `PUT /api/v1/projects/{id}/guidance` accepting the four levels and rejecting all other values;
- foreign Origin and missing Bearer token remaining rejected.

- [ ] **Step 2: Add failing React tests**

Use a fake `DashboardAPI` whose active guidance contains a high-severity redirect. Assert the first major heading is “任务守护中”, the current instruction is visible, the evidence disclosure remains collapsed, and the control-level selector calls `updateGuidanceControlLevel(projectId, 'guard')`.

- [ ] **Step 3: Run server and dashboard tests to confirm failure**

Run: `go test ./internal/server -run Guidance -count=1 && pnpm --dir dashboard test --run taskGuardian.test.tsx`

Expected: FAIL because the routes, types, and component do not exist.

- [ ] **Step 4: Implement authenticated guidance routes**

Extend the optional store interfaces without changing `EventStore`. Bound project IDs to 128 bytes, decode PUT bodies with `DisallowUnknownFields`, publish `guidance.settings.changed` through the existing hub after a successful update, and return generic errors without local paths.

- [ ] **Step 5: Implement dashboard types and API methods**

Add TypeScript equivalents of `GuidanceDecision` and `ControlLevel`, plus:

```ts
loadActiveGuidance(): Promise<GuidanceDecision[]>;
loadGuidanceControlLevel(projectId: string): Promise<{controlLevel: ControlLevel}>;
updateGuidanceControlLevel(projectId: string, controlLevel: ControlLevel): Promise<{controlLevel: ControlLevel}>;
```

Include active guidance in the existing refresh `Promise.all`; do not add polling because SSE already triggers refresh.

- [ ] **Step 6: Build the action-first TaskGuardian**

The component renders, in order:

1. current session/client/project safe label;
2. `正在推进`, `需要纠偏`, `等待验收`, or `已阻断` derived from kind;
3. one finding and one instruction;
4. evidence count and expiry;
5. control-level selector with short explanations;
6. recent intervention list, capped at five.

When the list is empty, render “当前没有需要介入的问题” and do not show a fake 100 score. Keep `AnalysisCockpit`, metrics, and raw transcript below the guardian as expandable evidence, not above it.

- [ ] **Step 7: Run dashboard, server, accessibility, and build checks**

Run: `go test ./internal/server -count=1 && pnpm --dir dashboard test --run && pnpm --dir dashboard build`

Expected: PASS; Vite emits `dist/index.html`, one CSS asset, and one JS asset.

- [ ] **Step 8: Commit the task control center**

```bash
git add internal/server dashboard
git commit -m "feat: make runtime guidance the dashboard primary action"
```

### Task 7: Embed, verify, and document capability boundaries

**Files:**
- Modify: `internal/server/web/index.html`
- Modify: `internal/server/web/assets/*`
- Modify: `README.md`
- Modify: `docs/usage.md`
- Modify: `docs/compatibility.md`
- Modify: `docs/privacy.md`
- Test: existing full suites

- [ ] **Step 1: Update user-facing capability statements**

Document Claude Code as Hook guidance/guard capable, Codex as MCP/Skill guidance capable, and all other adapters at their actual observed capability. Explicitly state that MCP/Skill guidance can be ignored by the client and is not a deterministic block. Document control levels and that raw prompts/tool content are not sent to an extra model.

- [ ] **Step 2: Rebuild and embed the dashboard**

Run: `./scripts/embed-dashboard.sh`

Expected: dashboard tests and build pass, then hashed assets are copied into `internal/server/web/assets` and `internal/server/web/index.html` references the new hashes.

- [ ] **Step 3: Run the complete verification suite**

Run:

```bash
go test ./... -race -count=1
go vet ./...
pnpm --dir dashboard test --run
pnpm --dir dashboard build
./scripts/check-secrets.sh
./scripts/check-docs.sh
git diff --check
```

Expected: every command exits 0; Go reports all packages `ok`, Vitest reports all test files passing, Vite completes a production build, and the hygiene scripts print no secret or documentation error.

- [ ] **Step 4: Run a local smoke test without opening a browser**

Run:

```bash
go build -o ./bin/agent-doctor ./cmd/agent-doctor
AGENT_DOCTOR_CONFIG_DIR="$(mktemp -d)" ./bin/agent-doctor start --once --no-open
```

Expected: command exits 0, prints a loopback dashboard URL, and does not bind a non-loopback interface.

- [ ] **Step 5: Commit documentation and embedded assets**

```bash
git add README.md docs/usage.md docs/compatibility.md docs/privacy.md internal/server/web
git commit -m "docs: explain Agent Doctor runtime guidance boundaries"
```

## Final review checklist

- [ ] Every emitted intervention references sanitized event IDs and an evidence fingerprint.
- [ ] The same evidence does not repeatedly inject the same guidance.
- [ ] Healthy sessions are quiet.
- [ ] Missing evidence never becomes a successful verification.
- [ ] Claude Hook failures remain fail-open and finish within the configured timeout.
- [ ] `guide` never blocks; `guard` and `autopilot` require explicit persisted selection.
- [ ] Codex is described as MCP/Skill guidance, not deterministic control.
- [ ] No prompt, source, raw command, tool output, credential, or absolute path enters guidance storage, MCP, HTTP, logs, or hook responses.
- [ ] The dashboard leads with tasks and interventions, with metrics and transcripts relegated to evidence.
