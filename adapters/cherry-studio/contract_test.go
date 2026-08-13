package cherrystudio

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCherryStudioUsesLocalReadOnlyMCP(t *testing.T) {
	raw, err := os.ReadFile("mcp.json.template")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document["command"] != "agent-doctor" {
		t.Fatalf("unexpected command: %#v", document)
	}
	arguments, ok := document["args"].([]any)
	if !ok || len(arguments) != 2 || arguments[0] != "mcp" || arguments[1] != "serve" {
		t.Fatalf("unexpected args: %#v", arguments)
	}
	if document["transport"] != "stdio" || document["readOnly"] != true {
		t.Fatalf("unsafe MCP declaration: %#v", document)
	}
}
