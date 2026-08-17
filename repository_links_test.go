package agentdoctor_test

import (
	"os"
	"strings"
	"testing"
)

func TestReadmesConnectNexoTokenDocsAndProduct(t *testing.T) {
	for _, filename := range []string{"README.md", "README.zh-CN.md"} {
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		text := string(content)
		for _, required := range []string{
			"https://www.nexotoken.net/official/tools/agent-doctor",
			"https://docs.nexotoken.net/",
			"https://docs.nexotoken.net/coding-tools/codex-cli/",
			"https://docs.nexotoken.net/coding-tools/claude-code/",
		} {
			if !strings.Contains(text, required) {
				t.Errorf("%s missing public link %s", filename, required)
			}
		}
		if strings.Contains(text, "18534516725.github.io/llm-api-setup-guides") {
			t.Errorf("%s still references the legacy docs origin", filename)
		}
	}
}
