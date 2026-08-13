package codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestNormalizeLifecycleEventAllowlistsMetadata(t *testing.T) {
	raw := json.RawMessage(`{
		"event_id":"evt-1","session_id":"session-1","project_id":"project-1",
		"timestamp":"2026-08-13T08:00:00Z","event":"session_start",
		"client_version":"1.2.3","model":"gpt-5","cwd":"/Users/alice/secret-project",
		"prompt":"do not store me","authorization":"Bearer secret-token"
	}`)

	event, err := NormalizeLifecycleEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != events.EventSessionStarted || event.Client.Name != "codex" {
		t.Fatalf("unexpected normalized event: %#v", event)
	}
	payload := string(event.Payload)
	for _, forbidden := range []string{"alice", "secret-project", "do not store me", "Bearer", "secret-token", "prompt", "authorization", "cwd"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("payload leaked %q: %s", forbidden, payload)
		}
	}
	if !strings.Contains(payload, `"hookEvent":"session_start"`) || !strings.Contains(payload, `"workingDirectoryHash":"sha256:`) {
		t.Fatalf("missing allowlisted lifecycle facts: %s", payload)
	}
	if err := events.Validate(event); err != nil {
		t.Fatalf("normalized event must satisfy the shared contract: %v", err)
	}
}

func TestNormalizeLifecycleEventRejectsUnsupportedOrIncompleteInput(t *testing.T) {
	for name, raw := range map[string]string{
		"unsupported event": `{"event_id":"e","session_id":"s","project_id":"p","timestamp":"2026-08-13T08:00:00Z","event":"raw_prompt"}`,
		"missing identity":  `{"timestamp":"2026-08-13T08:00:00Z","event":"session_start"}`,
		"invalid json":      `{`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeLifecycleEvent(json.RawMessage(raw)); err == nil {
				t.Fatal("expected normalization to fail")
			}
		})
	}
}

func TestIngestFailOpenNeverBlocksCodex(t *testing.T) {
	var log strings.Builder
	code := IngestFailOpen(strings.NewReader(`{"event":"unsupported"}`), func(events.Event) error {
		return nil
	}, &log)
	if code != 0 {
		t.Fatalf("hook adapter must fail open, got exit code %d", code)
	}
	if !strings.Contains(log.String(), "event was not recorded") || strings.Contains(log.String(), "unsupported") {
		t.Fatalf("expected a generic local diagnostic, got %q", log.String())
	}
}
