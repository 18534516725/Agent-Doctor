package guidance

import (
	"strings"
	"time"
)

type ConnectionState string

const (
	StateActive      ConnectionState = "active"
	StateObserving   ConnectionState = "observing"
	StateStale       ConnectionState = "stale"
	StateUnavailable ConnectionState = "unavailable"
	StateError       ConnectionState = "error"
)

type Status struct {
	State          ConnectionState `json:"state"`
	Client         string          `json:"client"`
	Advice         bool            `json:"advice"`
	Enforcement    bool            `json:"enforcement"`
	LastEvidenceAt *time.Time      `json:"lastEvidenceAt,omitempty"`
	Explanation    string          `json:"explanation"`
}

func ResolveStatus(lastEvidenceAt *time.Time, client string, activeDecision bool, now time.Time, repositoryError error) Status {
	if repositoryError != nil {
		return Status{State: StateError, Explanation: "guidance evidence could not be read"}
	}
	if lastEvidenceAt == nil {
		return Status{State: StateUnavailable, Explanation: "no compatible guidance evidence has been captured"}
	}
	client = strings.ToLower(strings.TrimSpace(client))
	status := Status{Client: client, Advice: client == "codex" || client == "claude-code", LastEvidenceAt: lastEvidenceAt}
	status.Enforcement = client == "claude-code"
	if now.Sub(lastEvidenceAt.UTC()) > 10*time.Minute {
		status.State = StateStale
		status.Explanation = "the most recent evidence is stale"
		return status
	}
	if activeDecision {
		status.State = StateActive
		status.Explanation = "an active guidance decision is available"
		return status
	}
	status.State = StateObserving
	status.Explanation = "evidence is arriving and no intervention is currently active"
	return status
}
