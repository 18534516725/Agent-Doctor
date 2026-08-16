package agentdoctorplugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginContract(t *testing.T) {
	manifest := readJSON(t, ".codex-plugin/plugin.json")
	for key, want := range map[string]string{
		"name": "agent-doctor", "version": "0.1.0", "license": "Apache-2.0",
		"repository": "https://github.com/18534516725/Agent-Doctor",
		"skills":     "./skills/", "mcpServers": "./.mcp.json",
	} {
		if manifest[key] != want {
			t.Fatalf("manifest %s = %#v, want %q", key, manifest[key], want)
		}
	}
	if _, exists := manifest["hooks"]; exists {
		t.Fatal("hooks must not be declared in plugin.json until the public contract supports them")
	}

	mcp := readJSON(t, ".mcp.json")
	servers := mcp["mcpServers"].(map[string]any)
	server := servers["agent-doctor"].(map[string]any)
	if server["command"] != "agent-doctor" {
		t.Fatalf("unexpected MCP command: %#v", server)
	}
	args := server["args"].([]any)
	if len(args) != 2 || args[0] != "mcp" || args[1] != "serve" {
		t.Fatalf("unexpected MCP arguments: %#v", args)
	}

	skill := readText(t, "skills/agent-doctor/SKILL.md")
	for _, required := range []string{"name: agent-doctor", "get_project_analysis", "before the final answer", "diagnose_last_task", "Never guess", "delivery receipts", "current project ID", "every coding turn", "Agent Doctor intervened"} {
		if !strings.Contains(skill, required) {
			t.Fatalf("skill is missing %q", required)
		}
	}
	if strings.Contains(skill, "TODO") {
		t.Fatal("skill contains an unfinished placeholder")
	}
	openAI := readText(t, "skills/agent-doctor/agents/openai.yaml")
	if !strings.Contains(openAI, "$agent-doctor") {
		t.Fatal("default prompt must explicitly invoke $agent-doctor")
	}
}

func readJSON(t *testing.T, name string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func readText(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
