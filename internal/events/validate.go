package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const MaxPayloadBytes = 256 * 1024

var supportedEventTypes = []string{
	EventSessionStarted,
	EventSessionResumed,
	EventSessionCompleted,
	EventSessionCancelled,
	EventUserPrompted,
	EventUserCorrected,
	EventToolStarted,
	EventToolCompleted,
	EventToolFailed,
	EventFileRead,
	EventFileChanged,
	EventCommandStarted,
	EventCommandCompleted,
	EventValidationCompleted,
	EventContextCompacted,
	EventContextInjected,
	EventModelChanged,
	EventUsageReported,
	EventCostReported,
	EventQuotaReported,
}

var eventTypeSet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(supportedEventTypes))
	for _, eventType := range supportedEventTypes {
		result[eventType] = struct{}{}
	}
	return result
}()

func SupportedEventTypes() []string {
	result := make([]string, len(supportedEventTypes))
	copy(result, supportedEventTypes)
	return result
}

func Validate(event Event) error {
	if event.SchemaVersion != 1 {
		return fmt.Errorf("unsupported event schema version %d", event.SchemaVersion)
	}
	if event.EventID == "" || event.SessionID == "" || event.ProjectID == "" {
		return errors.New("event, session, and project identities are required")
	}
	if event.Timestamp.IsZero() {
		return errors.New("event timestamp is required")
	}
	if event.Client.Name == "" || event.Client.Version == "" {
		return errors.New("client name and version are required")
	}
	if event.Model.DisplayName == "" {
		return errors.New("public model display name is required")
	}
	if _, ok := eventTypeSet[event.EventType]; !ok {
		return fmt.Errorf("unsupported event type %q", event.EventType)
	}
	if len(event.Payload) > MaxPayloadBytes {
		return fmt.Errorf("event payload exceeds %d bytes", MaxPayloadBytes)
	}
	if len(event.Payload) > 0 && !json.Valid(event.Payload) {
		return errors.New("event payload must be valid JSON")
	}
	if event.Provenance == "" {
		return errors.New("event provenance is required")
	}
	switch event.Precision {
	case PrecisionExact, PrecisionEstimated:
		return nil
	case PrecisionUnavailable:
		if !emptyPayload(event.Payload) {
			return errors.New("unavailable precision cannot include a claimed payload value")
		}
		return nil
	default:
		return fmt.Errorf("unsupported precision %q", event.Precision)
	}
}

func emptyPayload(payload json.RawMessage) bool {
	trimmed := bytes.TrimSpace(payload)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("{}"))
}
