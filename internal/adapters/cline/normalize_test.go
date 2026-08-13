package cline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestNormalizeClineHookDiscardsPromptsAndToolResults(t *testing.T) {
	raw := json.RawMessage(`{
		"taskId":"task-1","hookName":"PostToolUse","clineVersion":"3.36.0",
		"timestamp":"1786611600000","workspaceRoots":["/Users/alice/private-repo"],
		"model":{"provider":"private-provider","slug":"claude-sonnet"},
		"postToolUse":{"tool":"execute_command","parameters":{"command":"export API_KEY=secret"},
		"result":"private output","success":true,"durationMs":3450},"unknown":"do-not-copy"
	}`)
	event, err := NormalizeHook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != events.EventToolCompleted || event.Model.DisplayName != "claude-sonnet" {
		t.Fatalf("unexpected normalized event: %#v", event)
	}
	payload := string(event.Payload)
	for _, forbidden := range []string{"alice", "private-repo", "private-provider", "API_KEY", "secret", "private output", "parameters", "result", "unknown"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("payload leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{`"hookEvent":"PostToolUse"`, `"toolName":"execute_command"`, `"success":true`, `"durationMs":3450`} {
		if !strings.Contains(payload, required) {
			t.Fatalf("payload missing %q: %s", required, payload)
		}
	}
	if err := events.Validate(event); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeAllSupportedClineHooks(t *testing.T) {
	want := map[string]string{
		"TaskStart": events.EventSessionStarted, "TaskResume": events.EventSessionResumed,
		"TaskCancel": events.EventSessionCancelled, "TaskComplete": events.EventSessionCompleted,
		"UserPromptSubmit": events.EventUserPrompted, "PreCompact": events.EventContextCompacted,
		"PreToolUse": events.EventToolStarted, "PostToolUse": events.EventToolCompleted,
	}
	for hook, eventType := range want {
		raw := json.RawMessage(`{"taskId":"t","hookName":"` + hook + `","clineVersion":"3.36.0","timestamp":"1786611600000","workspaceRoots":["/repo"],"model":{"provider":"unknown","slug":"unknown"}}`)
		event, err := NormalizeHook(raw)
		if err != nil {
			t.Fatalf("%s: %v", hook, err)
		}
		if event.EventType != eventType {
			t.Fatalf("%s => %s", hook, event.EventType)
		}
	}
}

func TestClineContextResponseIsBoundedAndNeverCancels(t *testing.T) {
	response, err := ContextResponse(strings.Repeat("context ", 5000), 800)
	if err != nil {
		t.Fatal(err)
	}
	if response.Cancel || response.ErrorMessage != "" || len(response.ContextModification) > 3200 {
		t.Fatalf("unsafe response: %#v", response)
	}
}
