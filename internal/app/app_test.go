package app

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"version"}, &out, io.Discard)
	if code != 0 || !strings.Contains(out.String(), "agent-doctor dev") {
		t.Fatalf("unexpected result: code=%d output=%q", code, out.String())
	}
}

func TestUnknownCommandPrintsUsageToStderr(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"unknown"}, io.Discard, &stderr)
	if code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: agent-doctor <command>") {
		t.Fatalf("missing usage text: %q", stderr.String())
	}
}

func TestMCPServeStartsReadOnlyProtocolServer(t *testing.T) {
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}` + "\n")
	var out bytes.Buffer
	code := RunWithInput([]string{"mcp", "serve"}, input, &out, io.Discard)
	if code != 0 {
		t.Fatalf("code=%d output=%q", code, out.String())
	}
	for _, want := range []string{`"name":"agent-doctor"`, `"title":"Agent Doctor"`, `read-only`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("MCP response missing %q: %s", want, out.String())
		}
	}
}
