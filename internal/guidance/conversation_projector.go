package guidance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

const conversationProjectionProvenance = "codex-local-conversation-projection"

// ProjectConversation derives content-free operational evidence from a locally
// captured Codex request. Raw prompts, commands, paths and tool output never
// enter the returned events; only bounded labels and deterministic hashes do.
func ProjectConversation(record conversations.Request) []events.Event {
	if record.CompletedAt == nil || record.StatusCode == 0 {
		return nil
	}
	projected := make([]events.Event, 0, len(record.Messages)+1)
	failed := record.StatusCode >= 400
	for _, message := range record.Messages {
		if strings.TrimSpace(message.ToolName) == "" {
			continue
		}
		eventType := events.EventToolCompleted
		if failed {
			eventType = events.EventToolFailed
		}
		payload, _ := json.Marshal(eventFacts{
			ToolName:              boundedLabel(message.ToolName, 64),
			ToolInputFingerprint:  fingerprintJSON(message.ToolPayloadJSON),
			ToolResultFingerprint: fingerprintBytes([]byte(message.Content)),
		})
		projected = append(projected, projectedConversationEvent(
			record,
			stableProjectionID(record.ID, message.ID, eventType),
			eventType,
			messageTime(record, message),
			payload,
		))
	}
	if !failed && record.StatusCode >= 200 && record.StatusCode < 400 {
		payload, _ := json.Marshal(eventFacts{Status: "succeeded"})
		projected = append(projected, projectedConversationEvent(
			record,
			stableProjectionID(record.ID, "request", events.EventCommandCompleted),
			events.EventCommandCompleted,
			record.CompletedAt.UTC(),
			payload,
		))
	}
	return projected
}

func projectedConversationEvent(record conversations.Request, eventID, eventType string, at time.Time, payload []byte) events.Event {
	return events.Event{
		SchemaVersion: 1,
		EventID:       eventID,
		SessionID:     record.SessionID,
		ProjectID:     record.ProjectID,
		Timestamp:     at.UTC(),
		Client:        record.Client,
		Model:         record.Model,
		EventType:     eventType,
		Payload:       payload,
		Provenance:    conversationProjectionProvenance,
		Precision:     events.PrecisionExact,
	}
}

func messageTime(record conversations.Request, message conversations.Message) time.Time {
	if !message.CreatedAt.IsZero() {
		return message.CreatedAt.UTC()
	}
	if record.CompletedAt != nil {
		return record.CompletedAt.UTC()
	}
	return record.StartedAt.UTC()
}

func stableProjectionID(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "conversation-" + hex.EncodeToString(digest[:])
}

func fingerprintJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fingerprintBytes(nil)
	}
	var value any
	if json.Unmarshal([]byte(trimmed), &value) == nil {
		if canonical, err := json.Marshal(value); err == nil {
			return fingerprintBytes(canonical)
		}
	}
	return fingerprintBytes([]byte(trimmed))
}

func fingerprintBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func boundedLabel(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}
