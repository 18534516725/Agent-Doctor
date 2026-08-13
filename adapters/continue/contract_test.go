package continueadapter

import (
	"os"
	"strings"
	"testing"
)

func TestContinueConfigAddsOnlyLocalMCPDefinition(t *testing.T) {
	raw, err := os.ReadFile("config.yaml.template")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, required := range []string{"mcpServers:", "agent-doctor:", "command: agent-doctor", "- mcp"} {
		if !strings.Contains(text, required) {
			t.Fatalf("template missing %q", required)
		}
	}
}
