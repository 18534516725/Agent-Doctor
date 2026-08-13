package roocode

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRooCodeMCPTemplateUsesLocalStdio(t *testing.T) {
	raw, err := os.ReadFile("mcp_settings.json.template")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	servers := document["mcpServers"].(map[string]any)
	server := servers["agent-doctor"].(map[string]any)
	if server["command"] != "agent-doctor" {
		t.Fatalf("server=%#v", server)
	}
}
