package events

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func validEvent() Event {
	return Event{
		SchemaVersion: 1,
		EventID:       "event-1",
		SessionID:     "session-1",
		ProjectID:     "project-hash",
		Timestamp:     time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC),
		Client:        ClientRef{Name: "codex", Version: "1.0.0"},
		Model:         ModelRef{DisplayName: "public-model"},
		EventType:     EventSessionStarted,
		Payload:       json.RawMessage(`{"source":"client-event"}`),
		Provenance:    "client-event",
		Precision:     PrecisionExact,
	}
}

func TestValidateAcceptsKnownEvent(t *testing.T) {
	if err := Validate(validEvent()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnknownEventType(t *testing.T) {
	event := validEvent()
	event.EventType = "internal.raw"
	if err := Validate(event); err == nil || !strings.Contains(err.Error(), "event type") {
		t.Fatalf("expected event type error, got %v", err)
	}
}

func TestValidateRejectsUnsupportedSchema(t *testing.T) {
	event := validEvent()
	event.SchemaVersion = 2
	if err := Validate(event); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestValidateRejectsMissingIdentity(t *testing.T) {
	for _, field := range []string{"event", "session", "project"} {
		event := validEvent()
		switch field {
		case "event":
			event.EventID = ""
		case "session":
			event.SessionID = ""
		case "project":
			event.ProjectID = ""
		}
		if err := Validate(event); err == nil {
			t.Fatalf("expected missing %s identity to fail", field)
		}
	}
}

func TestValidateRejectsOversizedPayload(t *testing.T) {
	event := validEvent()
	event.Payload = json.RawMessage(`"` + strings.Repeat("x", MaxPayloadBytes) + `"`)
	if err := Validate(event); err == nil || !strings.Contains(err.Error(), "payload") {
		t.Fatalf("expected payload error, got %v", err)
	}
}

func TestValidateRejectsUnavailablePrecisionWithClaimedValue(t *testing.T) {
	event := validEvent()
	event.Precision = PrecisionUnavailable
	event.Payload = json.RawMessage(`{"tokenCount":42}`)
	if err := Validate(event); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable precision error, got %v", err)
	}
}

func TestAllDocumentedEventTypesValidate(t *testing.T) {
	for _, eventType := range SupportedEventTypes() {
		event := validEvent()
		event.EventType = eventType
		if err := Validate(event); err != nil {
			t.Fatalf("event type %q: %v", eventType, err)
		}
	}
}

func TestVersionedFixtureMatchesRuntimeContract(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/events/valid-session-started.json")
	if err != nil {
		t.Fatal(err)
	}

	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	if err := Validate(event); err != nil {
		t.Fatalf("fixture must satisfy runtime contract: %v", err)
	}
}

func TestValidateRequiresPublicClientAndModelIdentity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Event)
	}{
		{name: "client version", mutate: func(event *Event) { event.Client.Version = "" }},
		{name: "model display name", mutate: func(event *Event) { event.Model.DisplayName = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := validEvent()
			test.mutate(&event)
			if err := Validate(event); err == nil {
				t.Fatalf("expected missing %s to fail", test.name)
			}
		})
	}
}
