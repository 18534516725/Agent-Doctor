package claudecode

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

type hookInput struct {
	SessionID     string `json:"session_id"`
	WorkingDir    string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolUseID     string `json:"tool_use_id"`
	Source        string `json:"source"`
	Model         string `json:"model"`
}

var hookEventTypes = map[string]string{
	"SessionStart":       events.EventSessionStarted,
	"PreCompact":         events.EventContextCompacted,
	"PostToolUse":        events.EventToolCompleted,
	"PostToolUseFailure": events.EventToolFailed,
	"TaskCreated":        events.EventToolStarted,
	"TaskCompleted":      events.EventToolCompleted,
	"Stop":               events.EventSessionCompleted,
	"SessionEnd":         events.EventSessionCompleted,
}

// NormalizeHook accepts the documented Claude Code hook envelope while keeping
// only lifecycle metadata. Full transcripts, prompts, tool inputs, tool output,
// source code, credentials, and absolute paths are never copied to the event.
func NormalizeHook(raw json.RawMessage, receivedAt time.Time) (events.Event, error) {
	if len(raw) == 0 || len(raw) > maxHookInputBytes || receivedAt.IsZero() {
		return events.Event{}, fmt.Errorf("invalid hook envelope")
	}
	var input hookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return events.Event{}, fmt.Errorf("invalid hook envelope")
	}
	eventType, ok := hookEventTypes[input.HookEventName]
	if !ok || input.SessionID == "" || input.WorkingDir == "" {
		return events.Event{}, fmt.Errorf("unsupported or incomplete hook event")
	}
	projectDigest := sha256.Sum256([]byte(input.WorkingDir))
	projectID := "sha256:" + hex.EncodeToString(projectDigest[:])
	eventDigest := sha256.Sum256([]byte(strings.Join([]string{
		input.SessionID, input.HookEventName, input.ToolUseID, receivedAt.UTC().Format(time.RFC3339Nano),
	}, "\x00")))
	payload := map[string]string{
		"hookEvent":            input.HookEventName,
		"workingDirectoryHash": projectID,
	}
	if input.ToolName != "" {
		payload["toolName"] = boundedLabel(input.ToolName)
	}
	if input.Source != "" {
		payload["source"] = boundedLabel(input.Source)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return events.Event{}, fmt.Errorf("encode hook evidence")
	}
	model := boundedLabel(input.Model)
	if model == "" {
		model = "not-reported"
	}
	event := events.Event{
		SchemaVersion: 1,
		EventID:       "sha256:" + hex.EncodeToString(eventDigest[:]), SessionID: input.SessionID, ProjectID: projectID,
		Timestamp: receivedAt.UTC(), Client: events.ClientRef{Name: "claude-code", Version: "not-reported"},
		Model: events.ModelRef{DisplayName: model}, EventType: eventType, Payload: encoded,
		Provenance: "claude-code-official-hook", Precision: events.PrecisionExact,
	}
	if err := events.Validate(event); err != nil {
		return events.Event{}, fmt.Errorf("invalid normalized hook event")
	}
	return event, nil
}

func boundedLabel(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return value[:128]
	}
	return value
}

type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

type SessionStartOutput struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// SessionStartResponse emits only bounded context. It cannot make permission
// decisions, rewrite the user's prompt, or block Claude Code.
func SessionStartResponse(capsule string, tokenBudget int) (SessionStartOutput, error) {
	if tokenBudget < 1 || tokenBudget > 800 {
		return SessionStartOutput{}, fmt.Errorf("context budget must be between 1 and 800 tokens")
	}
	maxBytes := tokenBudget * 4
	capsule = strings.TrimSpace(capsule)
	if len(capsule) > maxBytes {
		capsule = capsule[:maxBytes]
	}
	return SessionStartOutput{HookSpecificOutput: HookSpecificOutput{
		HookEventName: "SessionStart", AdditionalContext: capsule,
	}}, nil
}

func IngestFailOpen(input io.Reader, receivedAt time.Time, store func(events.Event) error, diagnostics io.Writer) int {
	raw, err := io.ReadAll(io.LimitReader(input, maxHookInputBytes+1))
	if err == nil && len(raw) <= maxHookInputBytes {
		var event events.Event
		event, err = NormalizeHook(raw, receivedAt)
		if err == nil && store != nil {
			err = store(event)
		}
	}
	if err != nil || len(raw) > maxHookInputBytes || store == nil {
		if diagnostics != nil {
			fmt.Fprintln(diagnostics, "agent-doctor: local lifecycle event was not recorded; Claude Code will continue normally")
		}
	}
	return 0
}
