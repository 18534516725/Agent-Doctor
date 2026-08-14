package app

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
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
	var out bytes.Buffer
	code := Run([]string{"start", "--once"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "http://127.0.0.1:") || strings.Contains(strings.ToLower(out.String()), "browser opened") {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
}

func TestStartOnceAllocatesCaptureProxyWhenUpstreamIsConfigured(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("AGENT_DOCTOR_UPSTREAM_URL", "https://example.test")
	var out bytes.Buffer
	code := Run([]string{"start", "--once"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "proxy: http://127.0.0.1:") {
		t.Fatalf("code=%d output=%q", code, out.String())
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

type fakeLocalEvidenceStore struct{ events []events.Event }

func (store fakeLocalEvidenceStore) ListSessionEvents(_ context.Context, _ string) ([]events.Event, error) {
	return store.events, nil
}
