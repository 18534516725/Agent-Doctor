package guidance

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEvaluateRedirectsRepeatedIdenticalFailures(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{SessionID: "session-1", ProjectID: "project-1"}
	for i := 0; i < 3; i++ {
		state.Signals = append(state.Signals, Signal{
			EventID:           fmt.Sprintf("failure-%d", i+1),
			Kind:              SignalToolFailed,
			Tool:              "Bash",
			InputFingerprint:  "same-input",
			ResultFingerprint: "same-result",
			At:                now.Add(time.Duration(i) * time.Second),
		})
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindRedirect {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindRedirect)
	}
	if decision.Severity != SeverityHigh {
		t.Fatalf("severity = %q, want %q", decision.Severity, SeverityHigh)
	}
	if len(decision.Evidence) != 3 {
		t.Fatalf("evidence count = %d, want 3", len(decision.Evidence))
	}
}

func TestEvaluateAdvisesWhenReadsRepeatWithoutProgress(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{SessionID: "session-1", ProjectID: "project-1"}
	for i := 0; i < 4; i++ {
		state.Signals = append(state.Signals, Signal{
			EventID:          fmt.Sprintf("read-%d", i+1),
			Kind:             SignalToolCompleted,
			Tool:             "Read",
			InputFingerprint: "same-target",
			At:               now.Add(time.Duration(i) * time.Second),
		})
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindAdvise {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindAdvise)
	}
	if decision.Finding != "Repeated inspection without observable progress" {
		t.Fatalf("finding = %q", decision.Finding)
	}
}

func TestEvaluateAdvisesHandoffAfterContextCompaction(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{
		SessionID: "session-1",
		ProjectID: "project-1",
		Signals: []Signal{{
			EventID: "compact-1",
			Kind:    SignalContextCompacted,
			At:      now,
		}},
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindAdvise {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindAdvise)
	}
	if !strings.Contains(strings.ToLower(decision.Instruction), "preserve") {
		t.Fatalf("instruction %q does not preserve task state", decision.Instruction)
	}
}

func TestEvaluateRequiresValidationBeforeCompletion(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{
		SessionID: "session-1",
		ProjectID: "project-1",
		Signals:   []Signal{{EventID: "stop-1", Kind: SignalSessionCompleted, At: now}},
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindVerify {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindVerify)
	}
	if decision.Severity != SeverityHigh {
		t.Fatalf("severity = %q, want %q", decision.Severity, SeverityHigh)
	}
}

func TestEvaluateAllowsCompletionAfterSuccessfulValidation(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{
		SessionID: "session-1",
		ProjectID: "project-1",
		Signals: []Signal{
			{EventID: "change-1", Kind: SignalProgress, Progress: true, At: now},
			{EventID: "validation-1", Kind: SignalValidationPassed, At: now.Add(time.Second)},
			{EventID: "stop-1", Kind: SignalSessionCompleted, At: now.Add(2 * time.Second)},
		},
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindContinue {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindContinue)
	}
}

func TestEvaluateStaysQuietForHealthyProgress(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	state := SessionState{
		SessionID: "session-1",
		ProjectID: "project-1",
		Signals: []Signal{{
			EventID:  "change-1",
			Kind:     SignalProgress,
			Progress: true,
			At:       now,
		}},
	}

	decision := Evaluate(state, now.Add(time.Minute))
	if decision.Kind != KindContinue {
		t.Fatalf("kind = %q, want %q", decision.Kind, KindContinue)
	}
	if decision.Severity != SeverityInfo {
		t.Fatalf("severity = %q, want %q", decision.Severity, SeverityInfo)
	}
	if decision.Instruction != "" {
		t.Fatalf("instruction = %q, want quiet decision", decision.Instruction)
	}
}
