package claudecode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/guidance"
	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

const maxHookInputBytes = events.MaxPayloadBytes

type hookInput struct {
	SessionID     string          `json:"session_id"`
	WorkingDir    string          `json:"cwd"`
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolUseID     string          `json:"tool_use_id"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolResponse  json.RawMessage `json:"tool_response"`
	Source        string          `json:"source"`
	Model         string          `json:"model"`
}

var hookEventTypes = map[string]string{
	"SessionStart":       events.EventSessionStarted,
	"PreToolUse":         events.EventToolStarted,
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
	if fingerprint := fingerprintJSON(input.ToolInput); fingerprint != "" {
		payload["toolInputFingerprint"] = fingerprint
	}
	if fingerprint := fingerprintJSON(input.ToolResponse); fingerprint != "" {
		payload["toolResultFingerprint"] = fingerprint
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

func fingerprintJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func boundedLabel(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return value[:128]
	}
	return value
}

type HookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
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

type GuidanceResponse struct {
	Decision           string              `json:"decision,omitempty"`
	Reason             string              `json:"reason,omitempty"`
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

var (
	guidanceUnixPathPattern    = regexp.MustCompile(`(?:^|[\s(])/(?:[^/\s]+/)+[^\s),;]*`)
	guidanceWindowsPathPattern = regexp.MustCompile(`(?i)\b[A-Z]:\\(?:[^\\\r\n]+\\)*[^\s,;]*`)
)

// ResponseForDecision maps a content-free decision onto documented Claude
// hook response fields. Observe mode and stale or healthy decisions are silent.
func ResponseForDecision(hookName string, level guidance.ControlLevel, decision guidance.Decision, tokenBudget int, now time.Time) (GuidanceResponse, bool) {
	if tokenBudget < 1 || tokenBudget > 800 || level == guidance.ControlObserve || decision.Kind == guidance.KindContinue || !decision.ExpiresAt.After(now) {
		return GuidanceResponse{}, false
	}
	reason := boundedGuidanceText(strings.TrimSpace(strings.Join([]string{decision.Finding, decision.Instruction}, " ")), tokenBudget)
	if reason == "" {
		return GuidanceResponse{}, false
	}

	canEnforce := level == guidance.ControlGuard || level == guidance.ControlAutopilot
	if hookName == "PreToolUse" && decision.Kind == guidance.KindBlock && canEnforce {
		return GuidanceResponse{HookSpecificOutput: &HookSpecificOutput{
			HookEventName: hookName, PermissionDecision: "deny", PermissionDecisionReason: reason,
		}}, true
	}
	if hookName == "Stop" && decision.Kind == guidance.KindVerify && canEnforce {
		return GuidanceResponse{Decision: "block", Reason: reason}, true
	}
	if hookName == "SessionStart" || hookName == "PostToolUse" || hookName == "PostToolUseFailure" || hookName == "PreCompact" {
		switch decision.Kind {
		case guidance.KindAdvise, guidance.KindRedirect, guidance.KindVerify:
			return GuidanceResponse{HookSpecificOutput: &HookSpecificOutput{
				HookEventName: hookName, AdditionalContext: reason,
			}}, true
		}
	}
	return GuidanceResponse{}, false
}

func boundedGuidanceText(value string, tokenBudget int) string {
	value = privacy.FilterText(value)
	value = guidanceUnixPathPattern.ReplaceAllString(value, " [LOCAL_PATH]")
	value = guidanceWindowsPathPattern.ReplaceAllString(value, "[LOCAL_PATH]")
	limit := tokenBudget * 4
	if len(value) > limit {
		value = value[:limit]
	}
	return strings.TrimSpace(value)
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
