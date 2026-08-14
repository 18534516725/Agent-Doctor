# Live Editor Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Agent Doctor auto-connect supported local AI clients, capture complete model conversations through a loopback proxy, persist them in SQLite, stream live analysis to a redesigned Chinese-first dashboard, and preserve safe installation and removal.

**Architecture:** A Go loopback proxy records OpenAI- and Anthropic-compatible traffic without persisting authentication material, while official hooks continue to contribute editor lifecycle events. SQLite migration v4 stores messages, connection state, request timing, and analysis snapshots; a replayable SSE hub drives incremental React updates. Startup composes detection, previously consented automatic configuration, proxy, and dashboard services without making one client failure fatal to the process.

**Tech Stack:** Go 1.24, `modernc.org/sqlite`, `net/http`, React 19, TypeScript, Vite, Vitest, Testing Library, pnpm, GoReleaser.

---

## File map

- `migrations/004_live_observability.sql`: durable messages, requests, connections, analysis and consent state.
- `internal/conversations/types.go`: protocol-neutral message and request records.
- `internal/conversations/openai.go`: OpenAI request/response and SSE parsing.
- `internal/conversations/anthropic.go`: Anthropic request/response and SSE parsing.
- `internal/proxy/server.go`: authenticated loopback forwarding and capture lifecycle.
- `internal/proxy/credentials.go`: in-memory forwarding headers with prohibited persistence fields.
- `internal/storage/conversations.go`: transactional message/request/connection persistence and deletion.
- `internal/realtime/hub.go`: bounded replayable SSE event hub.
- `internal/server/routes.go`: conversation, connection, analysis and SSE APIs.
- `internal/server/server.go`: inject the event hub and expose live routes.
- `internal/installer/plan.go`: multi-client safe plans and installation consent marker.
- `internal/installer/managed.go`: format-preserving owned blocks/config entries.
- `internal/app/app.go`: compose auto-connect, proxy, capture and dashboard startup.
- `dashboard/src/i18n.ts`: Chinese-first interface copy.
- `dashboard/src/api.ts`: live API and SSE client contracts.
- `dashboard/src/App.tsx`: routed shell and live state composition.
- `dashboard/src/pages/*.tsx`: eight focused dashboard page components.
- `dashboard/src/components/*.tsx`: status rail, live conversation, metrics and connection components.
- `dashboard/src/styles.css`: refined instrument-panel visual system and motion.
- `docs/usage.md`, `docs/privacy.md`, `docs/compatibility.md`: exact setup, storage, capability and limits.

### Task 1: SQLite live-observability schema and repository

**Files:**
- Create: `migrations/004_live_observability.sql`
- Create: `internal/conversations/types.go`
- Create: `internal/storage/conversations.go`
- Modify: `internal/storage/storage_test.go`

- [ ] **Step 1: Write failing storage tests**

Add tests that open a temporary store, save a session request with complete user/assistant text, reload it in timestamp order, upsert a client connection, delete one session, and assert authorization/cookie/API-key-shaped transport headers never appear in any persisted column.

```go
func TestConversationRoundTripPreservesMessagesButNotCredentials(t *testing.T) {
    db := openTestStore(t)
    record := conversations.RequestRecord{
        ID: "request-1", SessionID: "session-1", Client: "codex",
        Messages: []conversations.Message{{Role: "user", Content: "explain this complete request"}},
        InputTokens: 12, OutputTokens: 8, Precision: "exact",
    }
    require.NoError(t, db.SaveConversation(context.Background(), record))
    got, err := db.Conversation(context.Background(), "session-1")
    require.NoError(t, err)
    require.Equal(t, "explain this complete request", got.Messages[0].Content)
    require.NotContains(t, dumpDatabaseText(t, db), "Bearer secret-value")
}
```

- [ ] **Step 2: Run the storage test and verify RED**

Run: `go test ./internal/storage -run 'TestConversation|TestClientConnection|TestDeleteConversation' -count=1`

Expected: FAIL because migration v4 and repository methods do not exist.

- [ ] **Step 3: Add migration v4 and minimal repository**

Create normalized tables `model_requests`, `messages`, `client_connections`, and `analysis_snapshots`. Use foreign keys with session deletion cascading to messages and request evidence. Store message bodies, tool JSON, usage precision and timing; omit every transport-auth field from the repository API.

```sql
CREATE TABLE messages (
  id TEXT PRIMARY KEY,
  request_id TEXT NOT NULL REFERENCES model_requests(id) ON DELETE CASCADE,
  session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  sequence INTEGER NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('system','user','assistant','tool')),
  content TEXT NOT NULL,
  tool_name TEXT NOT NULL DEFAULT '',
  tool_payload_json TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(request_id, sequence)
);
```

- [ ] **Step 4: Run storage tests and the migration suite**

Run: `go test ./internal/storage ./internal/events -race -count=1`

Expected: PASS with schema version 4.

- [ ] **Step 5: Commit**

```bash
git add migrations/004_live_observability.sql internal/conversations/types.go internal/storage/conversations.go internal/storage/storage_test.go
git commit -m "feat: persist live conversations locally"
```

### Task 2: OpenAI and Anthropic protocol capture

**Files:**
- Create: `internal/conversations/openai.go`
- Create: `internal/conversations/openai_test.go`
- Create: `internal/conversations/anthropic.go`
- Create: `internal/conversations/anthropic_test.go`

- [ ] **Step 1: Write failing protocol tests**

Use fixtures for Chat Completions, Responses, Anthropic Messages, OpenAI SSE and Anthropic SSE. Assert complete ordered text/tool messages and exact usage are reconstructed while unknown fields remain forward-compatible.

```go
func TestOpenAIStreamAssemblerReconstructsCompleteAssistantMessage(t *testing.T) {
    assembler := NewOpenAIStreamAssembler()
    assembler.Add([]byte(`data: {"choices":[{"delta":{"content":"hello "}}]}`))
    assembler.Add([]byte(`data: {"choices":[{"delta":{"content":"world"}}]}`))
    got := assembler.Complete()
    require.Equal(t, "hello world", got.Messages[0].Content)
}
```

- [ ] **Step 2: Run protocol tests and verify RED**

Run: `go test ./internal/conversations -count=1`

Expected: FAIL because the parsers do not exist.

- [ ] **Step 3: Implement bounded protocol parsers**

Use `json.Decoder`, explicit public protocol structs and byte limits. Preserve complete message text and tool JSON, but never accept request headers as parser input. Mark absent usage as unavailable rather than zero.

- [ ] **Step 4: Run protocol tests**

Run: `go test ./internal/conversations -race -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/conversations
git commit -m "feat: parse complete model conversations"
```

### Task 3: Loopback forwarding proxy

**Files:**
- Create: `internal/proxy/server.go`
- Create: `internal/proxy/server_test.go`
- Create: `internal/proxy/credentials.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing forwarding and secrecy tests**

Start an `httptest` upstream and the proxy on loopback. Verify method/path/query/body and streaming chunks reach the caller unchanged; verify the upstream receives Authorization while the store, events and logs never do. Verify cancellation closes the upstream request.

```go
func TestProxyForwardsAuthorizationInMemoryWithoutPersistingIt(t *testing.T) {
    // send Authorization to proxy, assert upstream receives it,
    // then assert captured RequestRecord has no header or credential field.
}
```

- [ ] **Step 2: Run proxy tests and verify RED**

Run: `go test ./internal/proxy ./internal/app -run 'TestProxy|TestStart' -count=1`

Expected: FAIL because the proxy package and composed startup are absent.

- [ ] **Step 3: Implement the loopback proxy**

Require a configured upstream base URL, reject non-HTTP(S) targets, prevent proxy recursion, use an explicit forwarding-header allowlist, strip hop-by-hop headers, stream through `io.Copy`, tee bounded protocol bytes into the matching assembler, and persist only after a successful database transaction.

- [ ] **Step 4: Compose proxy and dashboard startup**

Return both URLs in machine-readable startup state and human output. `--once` must allocate and close both listeners without leaving goroutines.

- [ ] **Step 5: Run proxy, app and full Go tests**

Run: `go test ./internal/proxy ./internal/app ./... -race -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/proxy internal/app
git commit -m "feat: capture model traffic through loopback proxy"
```

### Task 4: Replayable SSE and live APIs

**Files:**
- Create: `internal/realtime/hub.go`
- Create: `internal/realtime/hub_test.go`
- Modify: `internal/server/server.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/server_test.go`

- [ ] **Step 1: Write failing SSE and API tests**

Test monotonically increasing IDs, bounded replay, `Last-Event-ID`, heartbeat, slow-subscriber eviction, token/origin enforcement, conversation pagination, connection status and session deletion.

- [ ] **Step 2: Run server tests and verify RED**

Run: `go test ./internal/realtime ./internal/server -count=1`

Expected: FAIL because the hub and routes do not exist.

- [ ] **Step 3: Implement the hub and routes**

Publish only committed change notifications:

```go
type Event struct { ID uint64 `json:"id"`; Kind string `json:"kind"`; SessionID string `json:"sessionId,omitempty"` }
```

Expose `/api/v1/live`, `/api/v1/conversations`, `/api/v1/conversations/{id}`, `/api/v1/connections`, `/api/v1/analysis/live`, and DELETE for a session. Reuse existing session-token and Origin enforcement.

- [ ] **Step 4: Run server security and storage tests**

Run: `go test ./internal/realtime ./internal/server ./internal/storage -race -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/realtime internal/server internal/storage
git commit -m "feat: stream live diagnostic changes"
```

### Task 5: Safe automatic client connection

**Files:**
- Modify: `internal/installer/plan.go`
- Modify: `internal/installer/managed.go`
- Modify: `internal/installer/installer_test.go`
- Modify: `internal/app/app.go`
- Modify: `docs/compatibility.md`

- [ ] **Step 1: Write failing detection, consent and idempotence tests**

Cover a detected Codex config, JSON MCP clients, malformed config, concurrent external modification, backup restoration, previously granted auto-connect consent, restart-required status and unsupported clients.

- [ ] **Step 2: Run installer tests and verify RED**

Run: `go test ./internal/installer ./internal/app -run 'TestAuto|TestMultiClient|TestRollback' -count=1`

Expected: FAIL because setup currently owns only Codex.

- [ ] **Step 3: Implement conservative multi-client plans**

Automatically apply only adapters with a tested stable merge implementation. Preserve unknown JSON/TOML/YAML keys, write only Agent Doctor-owned entries, and record consent plus a backup manifest. Return capability states for every detected client.

- [ ] **Step 4: Connect startup to stored consent**

Formal installer executions grant auto-connect consent. Source builds without consent print the exact plan and continue with dashboard/proxy in limited mode; they never silently rewrite configs.

- [ ] **Step 5: Run installer and app suites**

Run: `go test ./internal/installer ./internal/app -race -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/installer internal/app docs/compatibility.md
git commit -m "feat: auto-connect supported local clients"
```

### Task 6: Chinese-first live dashboard foundation

**Files:**
- Create: `dashboard/src/i18n.ts`
- Create: `dashboard/src/live.ts`
- Create: `dashboard/src/components/StatusRail.tsx`
- Create: `dashboard/src/components/ConversationTimeline.tsx`
- Create: `dashboard/src/components/LiveMetrics.tsx`
- Create: `dashboard/src/components/ConnectionPanel.tsx`
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/tests/shell.test.tsx`
- Modify: `dashboard/tests/pages.test.tsx`

- [ ] **Step 1: Write failing dashboard behavior tests**

Assert Chinese is the initial language, user language persists, empty state explains listening status, all eleven clients show capability state, an SSE message incrementally adds a conversation item, reconnect retains the last event ID, and full user/assistant text renders as text rather than HTML.

- [ ] **Step 2: Run dashboard tests and verify RED**

Run: `pnpm --dir dashboard test --run`

Expected: FAIL on English default, static snapshots and missing live components.

- [ ] **Step 3: Implement focused components and live state**

Move copy into typed Chinese/English dictionaries, persist only `locale` in localStorage, inject a testable EventSource factory, update only affected slices, and retain an explicit reconnect/error banner.

- [ ] **Step 4: Run dashboard tests and TypeScript build**

Run: `pnpm --dir dashboard test --run && pnpm --dir dashboard build`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dashboard/src dashboard/tests
git commit -m "feat: add Chinese-first live dashboard state"
```

### Task 7: Refined instrument-panel visual system and eight pages

**Files:**
- Create: `dashboard/src/pages/OverviewPage.tsx`
- Create: `dashboard/src/pages/EvidencePage.tsx`
- Create: `dashboard/src/pages/CostsPage.tsx`
- Create: `dashboard/src/pages/MemoryPage.tsx`
- Create: `dashboard/src/pages/ComparisonPage.tsx`
- Create: `dashboard/src/pages/TrendsPage.tsx`
- Create: `dashboard/src/pages/IntegrationsPage.tsx`
- Create: `dashboard/src/pages/PrivacyPage.tsx`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/styles.css`
- Modify: `dashboard/tests/pages.test.tsx`

- [ ] **Step 1: Write failing semantic and resilience tests**

Assert each page has a unique heading and useful empty state, live metrics expose precision, conversation controls are keyboard accessible, long CJK/English text wraps, privacy deletion requires an explicit second action, and reduced motion is present in the stylesheet contract.

- [ ] **Step 2: Run tests and verify RED**

Run: `pnpm --dir dashboard test --run`

Expected: FAIL because the monolithic page and new visual contracts do not exist.

- [ ] **Step 3: Build the eight page components**

Use the approved precision-instrument direction: warm tinted surfaces, deep ink text, one signal color, asymmetric live overview, connection rail, full-width conversation timeline, real status labels, responsive container queries and no decorative fake charts.

- [ ] **Step 4: Add purposeful motion**

Use opacity/transform only for route entry, new-event insertion, connection changes and save/delete confirmation; make all motion effectively instant under `prefers-reduced-motion`.

- [ ] **Step 5: Run dashboard tests and production build**

Run: `pnpm --dir dashboard test --run && pnpm --dir dashboard build && ./scripts/embed-dashboard.sh`

Expected: PASS and deterministic embedded asset output.

- [ ] **Step 6: Commit**

```bash
git add dashboard internal/server/web
git commit -m "feat: redesign the real-time diagnostic workspace"
```

### Task 8: Privacy controls, retention and analysis correctness

**Files:**
- Modify: `internal/storage/conversations.go`
- Create: `internal/storage/retention.go`
- Create: `internal/storage/retention_test.go`
- Create: `internal/diagnostics/live.go`
- Create: `internal/diagnostics/live_test.go`
- Modify: `internal/server/routes.go`
- Modify: `dashboard/src/pages/PrivacyPage.tsx`
- Modify: `dashboard/src/pages/TrendsPage.tsx`

- [ ] **Step 1: Write failing deletion, retention and analysis tests**

Cover single-session deletion, project deletion, full deletion, expiry boundaries, exact/estimated/unavailable cost separation, latency percentiles, tool success, loop detection and missing-token behavior.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `go test ./internal/storage ./internal/diagnostics -run 'TestRetention|TestDelete|TestLive' -count=1`

Expected: FAIL because retention and live analysis do not exist.

- [ ] **Step 3: Implement transactional controls and deterministic analysis**

Delete cascades inside transactions, vacuum only from an explicit maintenance command, use UTC storage and local display formatting, and return evidence precision plus limitations with every analysis result.

- [ ] **Step 4: Run Go and dashboard tests**

Run: `go test ./internal/storage ./internal/diagnostics ./internal/server -race -count=1 && pnpm --dir dashboard test --run`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/storage internal/diagnostics internal/server dashboard/src/pages
git commit -m "feat: add local data controls and live analysis"
```

### Task 9: Documentation, release and no-browser verification

**Files:**
- Modify: `README.md`
- Modify: `docs/install.md`
- Modify: `docs/privacy.md`
- Modify: `docs/usage.md`
- Modify: `docs/compatibility.md`
- Create: `docs/test-reports/v1.1.0-live-observability.md`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Update documentation and CI contracts**

Document local proxy configuration, complete-message storage, credential exclusion, restart requirements, per-client capability and recovery. Add proxy/SSE/dashboard tests to CI on macOS, Windows and Linux.

- [ ] **Step 2: Run the complete release gate**

```bash
go test ./... -race -count=1
pnpm -r test --run
pnpm --dir dashboard build
./scripts/embed-dashboard.sh
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
```

Expected: all tests and checks pass.

- [ ] **Step 3: Run local HTTP/SSE lifecycle smoke tests without a browser**

Build `bin/agent-doctor`, start with `--no-open`, use `curl` to verify HTML, authenticated APIs, SSE heartbeat and proxy streaming, confirm both listeners are `127.0.0.1`, then stop cleanly.

- [ ] **Step 4: Rebuild six release archives and verify them**

Run the pinned GoReleaser/Syft pipeline, generate the release manifest, and run `scripts/verify-release.sh dist`.

- [ ] **Step 5: Write the evidence report and commit**

```bash
git add README.md docs .github/workflows/ci.yml internal/server/web
git commit -m "docs: verify live Agent Doctor release"
```

The report must distinguish local macOS runtime proof from cross-compiled Windows/Linux artifacts and must not claim a public GitHub release until the remote workflow succeeds.
