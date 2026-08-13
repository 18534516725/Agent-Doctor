package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryPublicCommandHasMachineReadableLocalBehavior(t *testing.T) {
	config := t.TempDir()
	home := t.TempDir()
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", config)
	t.Setenv("AGENT_DOCTOR_HOME", home)
	for _, command := range []string{"setup", "diagnose", "compare", "context", "costs", "doctor", "pause", "export", "uninstall"} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{command, "--json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%s code=%d stderr=%s", command, code, stderr.String())
		}
		if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "{") || strings.Contains(strings.ToLower(stdout.String()), "credential") {
			t.Fatalf("%s returned unsafe or non-JSON output: %s", command, stdout.String())
		}
	}
}

func TestSetupPlansBeforeApplyAndOnlyOwnsMarkedCodexBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AGENT_DOCTOR_HOME", home)
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	path := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("model = \"existing\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var preview bytes.Buffer
	if code := Run([]string{"setup", "--json"}, &preview, io.Discard); code != 0 {
		t.Fatalf("preview code=%d", code)
	}
	before, _ := os.ReadFile(path)
	if strings.Contains(string(before), "agent-doctor") {
		t.Fatal("preview modified configuration")
	}
	var applied bytes.Buffer
	if code := Run([]string{"setup", "--yes", "--json"}, &applied, io.Discard); code != 0 {
		t.Fatalf("apply code=%d output=%s", code, applied.String())
	}
	after, _ := os.ReadFile(path)
	if !strings.Contains(string(after), "mcp_servers.agent_doctor") || !strings.Contains(string(after), "model = \"existing\"") {
		t.Fatalf("configuration=%s", after)
	}
	if code := Run([]string{"uninstall", "--yes", "--json"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("uninstall code=%d", code)
	}
	restored, _ := os.ReadFile(path)
	if string(restored) != string(before) {
		t.Fatalf("user configuration not restored: %s", restored)
	}
}

func TestRunExecutesArgumentVectorWithoutShell(t *testing.T) {
	t.Setenv("AGENT_DOCTOR_CONFIG_DIR", t.TempDir())
	var stdout bytes.Buffer
	literal := "$(must-not-execute)"
	code := RunWithInput([]string{"run", "--", "printf", "%s", literal}, strings.NewReader(""), &stdout, io.Discard)
	if code != 0 || stdout.String() != literal {
		t.Fatalf("code=%d output=%q", code, stdout.String())
	}
}
