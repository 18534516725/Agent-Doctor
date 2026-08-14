package guidance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestProjectConversationProducesContentFreeFailureEvidence(t *testing.T) {
	started := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	completed := started.Add(time.Second)
	record := conversations.Request{
		ID: "request-1", SessionID: "session-1", ProjectID: "project-1",
		Client:     events.ClientRef{Name: "codex", Version: "1"},
		Model:      events.ModelRef{DisplayName: "gpt-test"},
		StatusCode: 500, StartedAt: started, CompletedAt: &completed,
		Messages: []conversations.Message{{
			ID: "message-1", Sequence: 1, Role: "tool", ToolName: "exec",
			ToolPayloadJSON: `{"path":"/secret/project","command":"rm private.txt"}`,
			Content:         "private failure output", CreatedAt: completed,
		}},
	}

	projected := ProjectConversation(record)
	if len(projected) != 1 || projected[0].EventType != events.EventToolFailed {
		t.Fatalf("projected=%+v", projected)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"/secret/project", "rm private.txt", "private failure output", "command", "path"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("projected evidence leaked %q: %s", secret, encoded)
		}
	}
	var facts eventFacts
	if err := json.Unmarshal(projected[0].Payload, &facts); err != nil {
		t.Fatal(err)
	}
	if facts.ToolName != "exec" || !strings.HasPrefix(facts.ToolInputFingerprint, "sha256:") || !strings.HasPrefix(facts.ToolResultFingerprint, "sha256:") {
		t.Fatalf("missing bounded evidence: %+v", facts)
	}
}

func TestProjectConversationCanonicalizesToolPayloadAndKeepsStableIDs(t *testing.T) {
	base := conversations.Request{
		ID: "request-stable", SessionID: "session-1", ProjectID: "project-1",
		Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "gpt-test"},
		StatusCode: 200, StartedAt: time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC),
		Messages: []conversations.Message{{ID: "message-stable", Role: "tool", ToolName: "exec", ToolPayloadJSON: `{"b":2,"a":1}`, Content: "ok"}},
	}
	completed := base.StartedAt.Add(time.Second)
	base.CompletedAt = &completed
	second := base
	second.Messages = append([]conversations.Message(nil), base.Messages...)
	second.Messages[0].ToolPayloadJSON = `{ "a": 1, "b": 2 }`

	firstEvents := ProjectConversation(base)
	secondEvents := ProjectConversation(second)
	if len(firstEvents) != 2 || len(secondEvents) != 2 {
		t.Fatalf("first=%+v second=%+v", firstEvents, secondEvents)
	}
	if firstEvents[0].EventID != secondEvents[0].EventID || firstEvents[1].EventID != secondEvents[1].EventID {
		t.Fatalf("event IDs changed: first=%+v second=%+v", firstEvents, secondEvents)
	}
	var firstFacts, secondFacts eventFacts
	_ = json.Unmarshal(firstEvents[0].Payload, &firstFacts)
	_ = json.Unmarshal(secondEvents[0].Payload, &secondFacts)
	if firstFacts.ToolInputFingerprint == "" || firstFacts.ToolInputFingerprint != secondFacts.ToolInputFingerprint {
		t.Fatalf("canonical fingerprints differ: %q != %q", firstFacts.ToolInputFingerprint, secondFacts.ToolInputFingerprint)
	}
	if firstEvents[1].EventType != events.EventCommandCompleted {
		t.Fatalf("successful request did not record progress: %+v", firstEvents)
	}
}
