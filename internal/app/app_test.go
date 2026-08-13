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
