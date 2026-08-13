package events

import (
	"encoding/json"
	"time"
)

type Precision string

const (
	PrecisionExact       Precision = "exact"
	PrecisionEstimated   Precision = "estimated"
	PrecisionUnavailable Precision = "unavailable"
)

const (
	EventSessionStarted      = "session.started"
	EventSessionResumed      = "session.resumed"
	EventSessionCompleted    = "session.completed"
	EventSessionCancelled    = "session.cancelled"
	EventUserPrompted        = "user.prompted"
	EventUserCorrected       = "user.corrected"
	EventToolStarted         = "tool.started"
	EventToolCompleted       = "tool.completed"
	EventToolFailed          = "tool.failed"
	EventFileRead            = "file.read"
	EventFileChanged         = "file.changed"
	EventCommandStarted      = "command.started"
	EventCommandCompleted    = "command.completed"
	EventValidationCompleted = "validation.completed"
	EventContextCompacted    = "context.compacted"
	EventContextInjected     = "context.injected"
	EventModelChanged        = "model.changed"
	EventUsageReported       = "usage.reported"
	EventCostReported        = "cost.reported"
	EventQuotaReported       = "quota.reported"
)

type ClientRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ModelRef struct {
	DisplayName string `json:"displayName"`
}

type Event struct {
	SchemaVersion int             `json:"schemaVersion"`
	EventID       string          `json:"eventId"`
	SessionID     string          `json:"sessionId"`
	ProjectID     string          `json:"projectId"`
	Timestamp     time.Time       `json:"timestamp"`
	Client        ClientRef       `json:"client"`
	Model         ModelRef        `json:"model"`
	EventType     string          `json:"eventType"`
	Payload       json.RawMessage `json:"payload"`
	Provenance    string          `json:"provenance"`
	Precision     Precision       `json:"precision"`
}
