package cursor

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCursorIntegrationUsesLocalStdioAndMinimalProjectRule(t *testing.T) {
	mcp := jsonObject(t, "mcp.json.template")
	servers, ok := mcp["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcp servers missing: %#v", mcp)
	}
	agentDoctor, ok := servers["agent-doctor"].(map[string]any)
	if !ok || agentDoctor["command"] != "agent-doctor" {
		t.Fatalf("unexpected MCP registration: %#v", agentDoctor)
	}
	arguments := agentDoctor["args"].([]any)
	if len(arguments) != 2 || arguments[0] != "mcp" || arguments[1] != "serve" {
		t.Fatalf("unexpected MCP args: %#v", arguments)
	}
	rule, err := os.ReadFile("rules/agent-doctor.mdc")
	if err != nil {
		t.Fatal(err)
	}
	text := string(rule)
	for _, required := range []string{"alwaysApply: false", "get_context_capsule", "Do not paste", "provenance"} {
		if !strings.Contains(text, required) {
			t.Fatalf("rule missing %q", required)
		}
	}
	if strings.Contains(text, "<full-history>") || strings.Contains(text, "Bearer ") {
		t.Fatal("rule must not contain embedded history or credentials")
	}
}

func jsonObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
