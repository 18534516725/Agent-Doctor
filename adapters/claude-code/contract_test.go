package claudecodeplugin

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestClaudePluginAndHookContract(t *testing.T) {
	manifest := object(t, ".claude-plugin/plugin.json")
	if manifest["name"] != "agent-doctor" || manifest["license"] != "Apache-2.0" {
		t.Fatalf("unexpected plugin manifest: %#v", manifest)
	}
	hookDocument := object(t, "hooks/hooks.json")
	hooks := hookDocument["hooks"].(map[string]any)
	wantEvents := []string{"SessionStart", "PreToolUse", "PreCompact", "PostToolUse", "PostToolUseFailure", "TaskCreated", "TaskCompleted", "Stop", "SessionEnd"}
	for _, eventName := range wantEvents {
		groups, ok := hooks[eventName].([]any)
		if !ok || len(groups) != 1 {
			t.Fatalf("missing hook %s", eventName)
		}
		handlers := groups[0].(map[string]any)["hooks"].([]any)
		handler := handlers[0].(map[string]any)
		if handler["type"] != "command" || handler["command"] != "agent-doctor" {
			t.Fatalf("%s must use the installed binary in exec form: %#v", eventName, handler)
		}
		if handler["timeout"].(float64) > 0.5 {
			t.Fatalf("%s timeout is not bounded: %#v", eventName, handler)
		}
		args := handler["args"].([]any)
		if len(args) != 3 || args[0] != "hook" || args[1] != "claude-code" || args[2] != eventName {
			t.Fatalf("unexpected args for %s: %#v", eventName, args)
		}
	}
	skill := text(t, "skills/agent-doctor/SKILL.md")
	for _, required := range []string{"delivery receipts", "Never guess", "provenance", "precision"} {
		if !strings.Contains(skill, required) {
			t.Fatalf("skill missing %q", required)
		}
	}
}

func object(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func text(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
