package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

const maxHookInputBytes = events.MaxPayloadBytes

type lifecycleInput struct {
	EventID       string `json:"event_id"`
	SessionID     string `json:"session_id"`
	ProjectID     string `json:"project_id"`
	Timestamp     string `json:"timestamp"`
	Event         string `json:"event"`
	ClientVersion string `json:"client_version"`
	Model         string `json:"model"`
	WorkingDir    string `json:"cwd"`
}

var lifecycleEventTypes = map[string]string{
	"session_start":    events.EventSessionStarted,
	"session_resume":   events.EventSessionResumed,
	"session_complete": events.EventSessionCompleted,
	"session_cancel":   events.EventSessionCancelled,
	"context_compact":  events.EventContextCompacted,
}

// NormalizeLifecycleEvent converts an allowlisted Codex lifecycle message into
// the shared event contract. Prompt text, source code, command output, absolute
// paths, credentials, and unknown fields are intentionally discarded.
func NormalizeLifecycleEvent(raw json.RawMessage) (events.Event, error) {
	if len(raw) == 0 || len(raw) > maxHookInputBytes {
		return events.Event{}, fmt.Errorf("invalid lifecycle message size")
	}
	var input lifecycleInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return events.Event{}, fmt.Errorf("invalid lifecycle message")
	}
	eventType, ok := lifecycleEventTypes[input.Event]
	if !ok {
		return events.Event{}, fmt.Errorf("unsupported lifecycle event")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, input.Timestamp)
	if err != nil {
		return events.Event{}, fmt.Errorf("invalid lifecycle timestamp")
	}
	if input.EventID == "" || input.SessionID == "" || input.ProjectID == "" {
		return events.Event{}, fmt.Errorf("lifecycle identities are required")
	}
	version := boundedLabel(input.ClientVersion, "not-reported")
	model := boundedLabel(input.Model, "not-reported")
	payload := map[string]string{"hookEvent": input.Event}
	if input.WorkingDir != "" {
		digest := sha256.Sum256([]byte(input.WorkingDir))
		payload["workingDirectoryHash"] = "sha256:" + hex.EncodeToString(digest[:])
	}
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return events.Event{}, fmt.Errorf("encode lifecycle evidence")
	}
	event := events.Event{
		SchemaVersion: 1,
		EventID:       input.EventID, SessionID: input.SessionID, ProjectID: input.ProjectID,
		Timestamp: timestamp.UTC(), Client: events.ClientRef{Name: "codex", Version: version},
		Model: events.ModelRef{DisplayName: model}, EventType: eventType,
		Payload: encodedPayload, Provenance: "codex-lifecycle-hook", Precision: events.PrecisionExact,
	}
	if err := events.Validate(event); err != nil {
		return events.Event{}, fmt.Errorf("invalid normalized lifecycle event")
	}
	return event, nil
}

func boundedLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if len(value) > 128 {
		return value[:128]
	}
	return value
}

// IngestFailOpen is the safe lifecycle adapter boundary. An unavailable local
// collector must never interrupt or change the user's Codex task.
func IngestFailOpen(input io.Reader, store func(events.Event) error, diagnostics io.Writer) int {
	limited := io.LimitReader(input, maxHookInputBytes+1)
	raw, err := io.ReadAll(limited)
	if err == nil && len(raw) <= maxHookInputBytes {
		var event events.Event
		event, err = NormalizeLifecycleEvent(raw)
		if err == nil && store != nil {
			err = store(event)
		}
	}
	if err != nil || len(raw) > maxHookInputBytes || store == nil {
		if diagnostics != nil {
			fmt.Fprintln(diagnostics, "agent-doctor: local lifecycle event was not recorded; Codex will continue normally")
		}
	}
	return 0
}
