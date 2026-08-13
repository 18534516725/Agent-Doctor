package aider

import (
	"os"
	"strings"
	"testing"
)

func TestAiderUsesOnlyTheArgvSafeWrapper(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, required := range []string{"agent-doctor run -- aider", "no transcript", "unavailable"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(required)) {
			t.Fatalf("Aider guidance missing %q", required)
		}
	}
	if strings.Contains(text, "sh -c") || strings.Contains(text, "Authorization:") {
		t.Fatal("Aider guidance contains an unsafe execution or credential example")
	}
}
