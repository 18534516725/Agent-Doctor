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
