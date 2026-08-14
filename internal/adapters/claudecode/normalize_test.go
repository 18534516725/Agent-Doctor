package claudecode

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestNormalizeOfficialHookAllowlistsEvidence(t *testing.T) {
	receivedAt := time.Date(2026, 8, 13, 9, 10, 11, 0, time.UTC)
	raw := json.RawMessage(`{
		"session_id":"session-1","cwd":"/Users/alice/secret-project",
		"hook_event_name":"PostToolUse","tool_name":"Bash","tool_use_id":"tool-1",
		"tool_input":{"command":"export API_KEY=secret && deploy"},
		"tool_response":{"content":"private output"},"transcript_path":"/private/transcript.jsonl"
	}`)
	event, err := NormalizeHook(raw, receivedAt)
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != events.EventToolCompleted || !event.Timestamp.Equal(receivedAt) {
		t.Fatalf("unexpected event: %#v", event)
	}
	payload := string(event.Payload)
	for _, forbidden := range []string{"alice", "secret-project", "API_KEY", "secret", "private output", "transcript", "tool_input", "tool_response"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("payload leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{`"hookEvent":"PostToolUse"`, `"toolName":"Bash"`, `"workingDirectoryHash":"sha256:`} {
		if !strings.Contains(payload, required) {
			t.Fatalf("payload missing %q: %s", required, payload)
		}
	}
	if err := events.Validate(event); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeHookFingerprintsCanonicalToolEvidence(t *testing.T) {
	receivedAt := time.Date(2026, 8, 13, 9, 10, 11, 0, time.UTC)
	first, err := NormalizeHook(json.RawMessage(`{
		"session_id":"session-1","cwd":"/repo","hook_event_name":"PostToolUse",
		"tool_name":"Read","tool_use_id":"tool-1",
		"tool_input":{"path":"private.go","options":{"limit":20,"offset":1}},
		"tool_response":{"status":"ok","lines":20}
	}`), receivedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NormalizeHook(json.RawMessage(`{
		"session_id":"session-1","cwd":"/repo","hook_event_name":"PostToolUse",
		"tool_name":"Read","tool_use_id":"tool-2",
		"tool_input":{"options":{"offset":1,"limit":20},"path":"private.go"},
		"tool_response":{"lines":20,"status":"ok"}
	}`), receivedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	var firstPayload, secondPayload map[string]string
	if err := json.Unmarshal(first.Payload, &firstPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Payload, &secondPayload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"toolInputFingerprint", "toolResultFingerprint"} {
		if firstPayload[key] == "" || !strings.HasPrefix(firstPayload[key], "sha256:") {
			t.Fatalf("%s = %q, want sha256 fingerprint", key, firstPayload[key])
		}
		if firstPayload[key] != secondPayload[key] {
			t.Fatalf("%s changed across equivalent JSON: %q != %q", key, firstPayload[key], secondPayload[key])
		}
	}
	encoded := string(first.Payload)
	for _, forbidden := range []string{"private.go", "options", "limit", "offset", "status", "lines"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("payload leaked tool evidence %q: %s", forbidden, encoded)
		}
	}
}

func TestNormalizeHookFingerprintChangesWithToolInput(t *testing.T) {
	receivedAt := time.Date(2026, 8, 13, 9, 10, 11, 0, time.UTC)
	normalize := func(path string) string {
		raw := json.RawMessage(`{"session_id":"session-1","cwd":"/repo","hook_event_name":"PostToolUse","tool_name":"Read","tool_input":{"path":"` + path + `"}}`)
		event, err := NormalizeHook(raw, receivedAt)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload["toolInputFingerprint"]
	}
	if normalize("one.go") == normalize("two.go") {
		t.Fatal("different tool inputs produced the same fingerprint")
	}
}

func TestNormalizeSupportedClaudeLifecycleEvents(t *testing.T) {
	want := map[string]string{
		"SessionStart":       events.EventSessionStarted,
		"PreCompact":         events.EventContextCompacted,
		"PostToolUse":        events.EventToolCompleted,
		"PostToolUseFailure": events.EventToolFailed,
		"TaskCreated":        events.EventToolStarted,
		"TaskCompleted":      events.EventToolCompleted,
		"Stop":               events.EventSessionCompleted,
		"SessionEnd":         events.EventSessionCompleted,
	}
	for hook, eventType := range want {
		raw := json.RawMessage(`{"session_id":"s","cwd":"/repo","hook_event_name":"` + hook + `"}`)
		event, err := NormalizeHook(raw, time.Unix(1, 0))
		if err != nil {
			t.Fatalf("%s: %v", hook, err)
		}
		if event.EventType != eventType {
			t.Fatalf("%s normalized to %s", hook, event.EventType)
		}
	}
}

func TestSessionStartResponseAddsOnlyBoundedCapsule(t *testing.T) {
	response, err := SessionStartResponse(strings.Repeat("context ", 5000), 800)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(response)
	if len(encoded) > 5000 || !strings.Contains(string(encoded), "additionalContext") {
		t.Fatalf("unexpected response size/content: %d %s", len(encoded), encoded)
	}
	if strings.Contains(string(encoded), "permissionDecision") || strings.Contains(string(encoded), "updatedPrompt") {
		t.Fatalf("context response must not control permissions or rewrite prompts: %s", encoded)
	}
}

func TestClaudeHookFailureIsFailOpen(t *testing.T) {
	var diagnostics strings.Builder
	if code := IngestFailOpen(strings.NewReader(`{"hook_event_name":"unknown"}`), time.Now(), nil, &diagnostics); code != 0 {
		t.Fatalf("hook must return 0, got %d", code)
	}
	if !strings.Contains(diagnostics.String(), "was not recorded") || strings.Contains(diagnostics.String(), "unknown") {
		t.Fatalf("unexpected diagnostics: %q", diagnostics.String())
	}
}
