package cline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type modelInput struct {
	Provider string `json:"provider"`
	Slug     string `json:"slug"`
}
type preToolInput struct {
	Tool string `json:"tool"`
}
type postToolInput struct {
	Tool       string `json:"tool"`
	Success    bool   `json:"success"`
	DurationMS int64  `json:"durationMs"`
}
type compactInput struct {
	ConversationLength int64 `json:"conversationLength"`
	EstimatedTokens    int64 `json:"estimatedTokens"`
}
type hookInput struct {
	TaskID         string        `json:"taskId"`
	HookName       string        `json:"hookName"`
	ClineVersion   string        `json:"clineVersion"`
	Timestamp      string        `json:"timestamp"`
	WorkspaceRoots []string      `json:"workspaceRoots"`
	Model          modelInput    `json:"model"`
	PreToolUse     preToolInput  `json:"preToolUse"`
	PostToolUse    postToolInput `json:"postToolUse"`
	PreCompact     compactInput  `json:"preCompact"`
}

var hookTypes = map[string]string{
	"TaskStart": events.EventSessionStarted, "TaskResume": events.EventSessionResumed,
	"TaskCancel": events.EventSessionCancelled, "TaskComplete": events.EventSessionCompleted,
	"UserPromptSubmit": events.EventUserPrompted, "PreCompact": events.EventContextCompacted,
	"PreToolUse": events.EventToolStarted, "PostToolUse": events.EventToolCompleted,
}

func NormalizeHook(raw json.RawMessage) (events.Event, error) {
	if len(raw) == 0 || len(raw) > events.MaxPayloadBytes {
		return events.Event{}, fmt.Errorf("invalid hook size")
	}
	var input hookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return events.Event{}, fmt.Errorf("invalid hook input")
	}
	eventType, ok := hookTypes[input.HookName]
	if !ok || input.TaskID == "" || len(input.WorkspaceRoots) == 0 || input.WorkspaceRoots[0] == "" {
		return events.Event{}, fmt.Errorf("unsupported or incomplete hook")
	}
	millis, err := strconv.ParseInt(input.Timestamp, 10, 64)
	if err != nil {
		return events.Event{}, fmt.Errorf("invalid hook timestamp")
	}
	projectDigest := sha256.Sum256([]byte(input.WorkspaceRoots[0]))
	projectID := "sha256:" + hex.EncodeToString(projectDigest[:])
	eventDigest := sha256.Sum256([]byte(input.TaskID + "\x00" + input.HookName + "\x00" + input.Timestamp))
	payload := map[string]any{"hookEvent": input.HookName, "workingDirectoryHash": projectID}
	switch input.HookName {
	case "PreToolUse":
		if input.PreToolUse.Tool != "" {
			payload["toolName"] = bounded(input.PreToolUse.Tool)
		}
	case "PostToolUse":
		if input.PostToolUse.Tool != "" {
			payload["toolName"] = bounded(input.PostToolUse.Tool)
		}
		payload["success"] = input.PostToolUse.Success
		if input.PostToolUse.DurationMS >= 0 {
			payload["durationMs"] = input.PostToolUse.DurationMS
		}
	case "PreCompact":
		if input.PreCompact.ConversationLength >= 0 {
			payload["conversationLength"] = input.PreCompact.ConversationLength
		}
		if input.PreCompact.EstimatedTokens >= 0 {
			payload["estimatedTokens"] = input.PreCompact.EstimatedTokens
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return events.Event{}, fmt.Errorf("encode hook evidence")
	}
	model := bounded(input.Model.Slug)
	if model == "" || model == "unknown" {
		model = "not-reported"
	}
	version := bounded(input.ClineVersion)
	if version == "" {
		version = "not-reported"
	}
	event := events.Event{SchemaVersion: 1, EventID: "sha256:" + hex.EncodeToString(eventDigest[:]), SessionID: input.TaskID, ProjectID: projectID,
		Timestamp: time.UnixMilli(millis).UTC(), Client: events.ClientRef{Name: "cline", Version: version}, Model: events.ModelRef{DisplayName: model},
		EventType: eventType, Payload: encoded, Provenance: "cline-official-hook", Precision: events.PrecisionExact}
	if err := events.Validate(event); err != nil {
		return events.Event{}, fmt.Errorf("invalid normalized hook")
	}
	return event, nil
}

func bounded(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return value[:128]
	}
	return value
}

type HookResponse struct {
	Cancel              bool   `json:"cancel"`
	ContextModification string `json:"contextModification,omitempty"`
	ErrorMessage        string `json:"errorMessage,omitempty"`
}

func ContextResponse(capsule string, budget int) (HookResponse, error) {
	if budget < 1 || budget > 800 {
		return HookResponse{}, fmt.Errorf("context budget must be between 1 and 800 tokens")
	}
	capsule = strings.TrimSpace(capsule)
	maxBytes := budget * 4
	if len(capsule) > maxBytes {
		capsule = capsule[:maxBytes]
	}
	return HookResponse{Cancel: false, ContextModification: capsule}, nil
}
