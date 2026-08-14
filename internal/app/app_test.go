package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/guidance"
)

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"version"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "agent-doctor dev") {
		t.Fatalf("unexpected result: code=%d output=%q", code, out.String())
	}
}

func TestUnknownCommandPrintsUsageToStderr(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"unknown"}, io.Discard, &stderr)
	if code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: agent-doctor <command>") || !strings.Contains(stderr.String(), "setup") || !strings.Contains(stderr.String(), "dashboard") {
		t.Fatalf("missing usage text: %q", stderr.String())
	}
}

func TestStartOncePrintsLocalDashboardURLWithoutOpeningABrowser(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("AGENT_DOCTOR_HOME", t.TempDir())
	var out bytes.Buffer
	code := Run([]string{"start", "--once"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "http://127.0.0.1:") || strings.Contains(strings.ToLower(out.String()), "browser opened") {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
}

func TestStartOnceAllocatesCaptureProxyWhenUpstreamIsConfigured(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("AGENT_DOCTOR_HOME", t.TempDir())
	t.Setenv("AGENT_DOCTOR_UPSTREAM_URL", "https://example.test")
	var out bytes.Buffer
	code := Run([]string{"start", "--once"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "proxy: http://127.0.0.1:") {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
}

func TestPrepareManagedIntegrationsInstallsCodexBlockIdempotently(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("model = \"existing\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := prepareManagedIntegrations(home)
	if err != nil {
		t.Fatal(err)
	}
	second, err := prepareManagedIntegrations(home)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if first.Applied != 1 || second.Applied != 0 || second.Skipped != 1 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	text := string(contents)
	if strings.Count(text, "# >>> agent-doctor:codex >>>") != 1 || !strings.Contains(text, "model = \"existing\"") {
		t.Fatalf("unexpected config: %s", text)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `command = "`+strings.ReplaceAll(executable, `\`, `\\`)+`"`) {
		t.Fatalf("managed MCP command does not use current executable %q: %s", executable, text)
	}
}

func TestBrowserCommandUsesArgumentVectorsWithoutShell(t *testing.T) {
	url := "http://127.0.0.1:51993/?value=$(must-not-run)"
	tests := []struct {
		goos string
		name string
		args []string
	}{
		{goos: "darwin", name: "open", args: []string{url}},
		{goos: "linux", name: "xdg-open", args: []string{url}},
		{goos: "windows", name: "rundll32", args: []string{"url.dll,FileProtocolHandler", url}},
	}
	for _, test := range tests {
		name, args, err := browserCommand(test.goos, url)
		if err != nil {
			t.Fatalf("%s: %v", test.goos, err)
		}
		if name != test.name || strings.Join(args, "\x00") != strings.Join(test.args, "\x00") {
			t.Fatalf("%s command=%q args=%q", test.goos, name, args)
		}
	}
}

func TestBrowserCommandRejectsUnsupportedPlatforms(t *testing.T) {
	if _, _, err := browserCommand("plan9", "http://127.0.0.1:51993/"); err == nil {
		t.Fatal("unsupported platform was accepted")
	}
}

func TestStartBrowserOptOutContract(t *testing.T) {
	if !shouldOpenBrowser(nil) {
		t.Fatal("plain start should open the dashboard")
	}
	if shouldOpenBrowser([]string{"--no-open"}) {
		t.Fatal("--no-open should disable browser launch")
	}
	if shouldOpenBrowser([]string{"--once"}) {
		t.Fatal("--once should never launch a browser")
	}
}

func TestBrowserEnvironmentDoesNotForwardCredentials(t *testing.T) {
	environment := browserEnvironment([]string{
		"PATH=/usr/bin",
		"HOME=/tmp/home",
		"DISPLAY=:0",
		"OPENAI_API_KEY=must-not-forward",
		"GH_TOKEN=must-not-forward",
		"SESSION_SECRET=must-not-forward",
	})
	joined := strings.Join(environment, "\n")
	for _, allowed := range []string{"PATH=/usr/bin", "HOME=/tmp/home", "DISPLAY=:0"} {
		if !strings.Contains(joined, allowed) {
			t.Fatalf("missing safe environment %q in %q", allowed, joined)
		}
	}
	for _, forbidden := range []string{"must-not-forward", "OPENAI_API_KEY", "GH_TOKEN", "SESSION_SECRET"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("browser environment leaked %q: %q", forbidden, joined)
		}
	}
}

func TestCaptureUpstreamPrefersExplicitAgentDoctorConfiguration(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://anthropic.example")
	t.Setenv("OPENAI_BASE_URL", "https://openai.example")
	t.Setenv("AGENT_DOCTOR_UPSTREAM_URL", "https://explicit.example")
	if got := captureUpstreamURL(); got != "https://explicit.example" {
		t.Fatalf("captureUpstreamURL()=%q", got)
	}
}

func TestOverriddenEnvironmentDoesNotDuplicateBaseURL(t *testing.T) {
	environment := overriddenEnvironment([]string{"PATH=/bin", "OPENAI_BASE_URL=https://old.example"}, map[string]string{"OPENAI_BASE_URL": "http://127.0.0.1:4321"})
	joined := strings.Join(environment, "\n")
	if strings.Count(joined, "OPENAI_BASE_URL=") != 1 || !strings.Contains(joined, "OPENAI_BASE_URL=http://127.0.0.1:4321") {
		t.Fatalf("environment=%q", environment)
	}
}

func TestDoctorJSONReportsLocalInstallationState(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	var out bytes.Buffer
	code := Run([]string{"doctor", "--json"}, &out, io.Discard)
	if code != 0 {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
	for _, expected := range []string{`"status":"ready"`, `"database"`, `"detectedClients"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("doctor output missing %s: %s", expected, out.String())
		}
	}
}

func TestMCPServeStartsReadOnlyProtocolServer(t *testing.T) {
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}` + "\n")
	var out bytes.Buffer
	code := RunWithInput([]string{"mcp", "serve"}, input, &out, io.Discard)
	if code != 0 {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
	for _, want := range []string{`"name":"agent-doctor"`, `"title":"Agent Doctor"`, `read-only`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("MCP response missing %q: %s", want, out.String())
		}
	}
}

func TestCodexMCPServePublishesConnectionLifecycle(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	reader, writer := io.Pipe()
	var output bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- RunWithInput([]string{"mcp", "serve"}, reader, &output, io.Discard)
	}()
	if _, err := io.WriteString(writer, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"codex","version":"test"}}}`+"\n"); err != nil {
		t.Fatal(err)
	}

	waitForCodexConnectionState(t, "connected")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("mcp serve code=%d output=%s", code, output.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("mcp serve did not stop after stdin closed")
	}
	waitForCodexConnectionState(t, "detected")
}

func waitForCodexConnectionState(t *testing.T, expected string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		database, err := openLocalStore()
		if err == nil {
			connections, listErr := database.ListClientConnections(context.Background())
			_ = database.Close()
			if listErr == nil {
				for _, connection := range connections {
					if connection.Key == "codex" && connection.State == expected {
						return
					}
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("codex connection never reached %q", expected)
}

func TestClaudeCodeHookRecordsOnlyNormalizedLifecycleEvidence(t *testing.T) {
	input := strings.NewReader(`{"session_id":"session-1","cwd":"/private/project","hook_event_name":"PreCompact","model":"public-model","tool_input":{"secret":"must-not-store"}}`)
	var captured events.Event
	code := runClaudeCodeHook(input, func(event events.Event) error {
		captured = event
		return nil
	}, io.Discard, func() time.Time { return time.Date(2026, 8, 13, 14, 0, 0, 0, time.UTC) })
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if captured.EventType != events.EventContextCompacted || captured.Client.Name != "claude-code" {
		t.Fatalf("event=%+v", captured)
	}
	encoded := string(captured.Payload)
	for _, forbidden := range []string{"private/project", "must-not-store", "tool_input"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("normalized hook leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestClineHookRecordsOnlyNormalizedLifecycleEvidence(t *testing.T) {
	input := strings.NewReader(`{"taskId":"task-1","hookName":"PreCompact","clineVersion":"1.2.3","timestamp":"1786600800000","workspaceRoots":["/private/project"],"model":{"provider":"private","slug":"public-model"},"preCompact":{"conversationLength":10,"estimatedTokens":100},"transcript":"must-not-store"}`)
	var captured events.Event
	code := runClineHook(input, func(event events.Event) error {
		captured = event
		return nil
	}, io.Discard)
	if code != 0 || captured.EventType != events.EventContextCompacted || captured.Client.Name != "cline" {
		t.Fatalf("code=%d event=%+v", code, captured)
	}
	encoded := string(captured.Payload)
	for _, forbidden := range []string{"private/project", "must-not-store", "private\""} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("normalized hook leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestLocalMCPBackendReturnsSanitizedTaskEvidence(t *testing.T) {
	backend := localMCPBackend{store: fakeLocalEvidenceStore{events: []events.Event{{
		SessionID: "session-1", EventType: events.EventContextCompacted,
		Timestamp: time.Date(2026, 8, 13, 14, 0, 0, 0, time.UTC),
		Payload:   []byte(`{"workingDirectoryHash":"sha256:example","prompt":"must-not-leak"}`),
		Precision: events.PrecisionExact,
	}}}}
	evidence, err := backend.Execute(context.Background(), "get_task_evidence", map[string]any{"sessionId": "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Precision != "exact" || len(evidence.Items) != 1 || !strings.Contains(evidence.Items[0].Value, "context.compacted") {
		t.Fatalf("evidence=%+v", evidence)
	}
	encoded := evidence.Summary + " " + evidence.Items[0].Label + " " + evidence.Items[0].Value
	for _, forbidden := range []string{"must-not-leak", "workingDirectoryHash", "sha256:example"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("MCP evidence leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestLocalMCPBackendReturnsCurrentProjectAnalysis(t *testing.T) {
	backend := localMCPBackend{store: fakeLocalAnalysisStore{
		fakeLocalEvidenceStore: fakeLocalEvidenceStore{},
		analysis: conversations.LiveAnalysis{
			ProjectID: "project-current", Requests: 4, FailedRequests: 1, HealthScore: 72,
			TokensPerRequest: 12_500, CacheHitRate: 20, AverageLatencyMS: 2_500,
			Summary:     "当前项目需要处理一个失败请求。",
			Findings:    []conversations.Finding{{Title: "失败请求需要处理", Evidence: "1/4 次失败", Recommendation: "先查看失败证据"}},
			Limitations: []string{"费用目录不完整。"},
		},
	}}
	evidence, err := backend.Execute(context.Background(), "get_project_analysis", nil)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Summary != "当前项目需要处理一个失败请求。" || evidence.Provenance != "local-sqlite-deterministic-analysis" {
		t.Fatalf("analysis evidence=%+v", evidence)
	}
	joined := evidence.Summary
	for _, item := range evidence.Items {
		joined += " " + item.Label + " " + item.Value
	}
	for _, expected := range []string{"72/100", "4 次请求，1 次失败", "1/4 次失败", "先查看失败证据"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("analysis evidence missing %q: %s", expected, joined)
		}
	}
}

func TestLocalMCPBackendReturnsRuntimeGuidance(t *testing.T) {
	now := time.Now().UTC()
	backend := localMCPBackend{store: fakeLocalGuidanceStore{
		fakeLocalEvidenceStore: fakeLocalEvidenceStore{events: []events.Event{
			{EventID: "event-1", Precision: events.PrecisionExact},
			{EventID: "event-2", Precision: events.PrecisionExact},
			{EventID: "event-3", Precision: events.PrecisionExact},
		}},
		decision: guidance.Decision{
			DecisionID: "decision-1", SessionID: "session-1", ProjectID: "project-1",
			Kind: guidance.KindRedirect, Severity: guidance.SeverityHigh,
			Finding:           "The same tool action is failing repeatedly",
			Evidence:          []string{"event-1", "event-2", "event-3"},
			Instruction:       "Choose a different diagnostic step.",
			ProhibitedActions: []string{"Do not repeat the identical action."},
			Verification:      []string{"Run a deterministic check."},
			CreatedAt:         now, ExpiresAt: now.Add(time.Minute),
		},
		level: guidance.ControlGuide,
	}}
	evidence, err := backend.Execute(context.Background(), "get_runtime_guidance", map[string]any{"sessionId": "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	joined := evidence.Summary
	for _, item := range evidence.Items {
		joined += " " + item.Label + " " + item.Value
	}
	for _, expected := range []string{"failing repeatedly", "redirect", "high", "Choose a different", "event-1", "guide"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("guidance missing %q: %+v", expected, evidence)
		}
	}
	if evidence.Provenance != "local-sqlite-deterministic-guidance" || evidence.Precision != "exact" {
		t.Fatalf("guidance metadata=%+v", evidence)
	}
}

type fakeLocalEvidenceStore struct{ events []events.Event }

func (store fakeLocalEvidenceStore) ListSessionEvents(_ context.Context, _ string) ([]events.Event, error) {
	return store.events, nil
}

type fakeLocalAnalysisStore struct {
	fakeLocalEvidenceStore
	analysis conversations.LiveAnalysis
}

type fakeLocalGuidanceStore struct {
	fakeLocalEvidenceStore
	decision guidance.Decision
	level    guidance.ControlLevel
}

func (store fakeLocalGuidanceStore) RuntimeGuidance(_ context.Context, _ string, _ time.Time) (guidance.Decision, error) {
	return store.decision, nil
}

func (store fakeLocalGuidanceStore) LatestRuntimeGuidance(_ context.Context, _ string, _ time.Time) (guidance.Decision, error) {
	return store.decision, nil
}

func (store fakeLocalGuidanceStore) GuidanceControlLevel(_ context.Context, _ string) (guidance.ControlLevel, error) {
	return store.level, nil
}

func (store fakeLocalAnalysisStore) LiveConversationAnalysis(_ context.Context) (conversations.LiveAnalysis, error) {
	return store.analysis, nil
}
