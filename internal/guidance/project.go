package guidance

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type eventFacts struct {
	HookEventName         string `json:"hookEvent"`
	ToolName              string `json:"toolName"`
	ToolInputFingerprint  string `json:"toolInputFingerprint"`
	ToolResultFingerprint string `json:"toolResultFingerprint"`
	Status                string `json:"status"`
}

// Project converts normalized telemetry into content-free guidance signals.
func Project(source []events.Event) SessionState {
	state := SessionState{}
	for _, event := range source {
		if event.SessionID != "" {
			state.SessionID = event.SessionID
		}
		if event.ProjectID != "" {
			state.ProjectID = event.ProjectID
		}

		var facts eventFacts
		_ = json.Unmarshal(event.Payload, &facts)
		signal := Signal{
			EventID:           event.EventID,
			Tool:              facts.ToolName,
			InputFingerprint:  facts.ToolInputFingerprint,
			ResultFingerprint: facts.ToolResultFingerprint,
			At:                event.Timestamp,
		}
		switch event.EventType {
		case events.EventToolFailed:
			signal.Kind = SignalToolFailed
		case events.EventToolCompleted:
			signal.Kind = SignalToolCompleted
			signal.Progress = isProgressTool(facts.ToolName) || strings.EqualFold(facts.HookEventName, "TaskCompleted")
		case events.EventFileChanged, events.EventCommandCompleted:
			signal.Kind = SignalProgress
			signal.Progress = true
		case events.EventValidationCompleted:
			if !isPassedStatus(facts.Status) {
				continue
			}
			signal.Kind = SignalValidationPassed
		case events.EventContextCompacted:
			signal.Kind = SignalContextCompacted
		case events.EventSessionCompleted:
			signal.Kind = SignalSessionCompleted
		default:
			continue
		}
		state.Signals = append(state.Signals, signal)
	}
	sort.SliceStable(state.Signals, func(i, j int) bool {
		if state.Signals[i].At.Equal(state.Signals[j].At) {
			return state.Signals[i].EventID < state.Signals[j].EventID
		}
		return state.Signals[i].At.Before(state.Signals[j].At)
	})
	return state
}

func isProgressTool(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "edit", "write", "notebookedit", "taskcompleted":
		return true
	default:
		return false
	}
}

func isPassedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "success", "succeeded":
		return true
	default:
		return false
	}
}
