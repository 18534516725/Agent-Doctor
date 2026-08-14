package guidance

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const decisionTTL = 10 * time.Minute

func Evaluate(state SessionState, now time.Time) Decision {
	decision := Decision{
		SessionID:         state.SessionID,
		ProjectID:         state.ProjectID,
		Kind:              KindContinue,
		Severity:          SeverityInfo,
		Evidence:          []string{},
		Confidence:        "supported",
		ProhibitedActions: []string{},
		Verification:      []string{},
		CreatedAt:         now,
		ExpiresAt:         now,
	}

	if len(state.Signals) == 0 {
		return finalize(decision)
	}

	if evidence, needsValidation := completionNeedsValidation(state.Signals); needsValidation {
		decision.Kind = KindVerify
		decision.Severity = SeverityHigh
		decision.Finding = "Completion has not been supported by a successful validation"
		decision.Evidence = evidence
		decision.Instruction = "Run an approved deterministic validation before claiming the task is complete."
		decision.ProhibitedActions = []string{"Do not claim completion without validation evidence."}
		decision.Verification = []string{"Run the relevant test, build, lint, or deterministic check."}
		decision.ExpiresAt = now.Add(decisionTTL)
		return finalize(decision)
	}

	if evidence, repeated := trailingRepeatedFailures(state.Signals); repeated {
		decision.Kind = KindRedirect
		decision.Severity = SeverityHigh
		decision.Finding = "The same tool action is failing repeatedly"
		decision.Evidence = evidence
		decision.Instruction = "Stop repeating this tool call. Inspect the unchanged failure evidence and choose a different diagnostic step before retrying."
		decision.ProhibitedActions = []string{"Do not retry the identical action without a changed hypothesis or input."}
		decision.ExpiresAt = now.Add(decisionTTL)
		return finalize(decision)
	}

	last := state.Signals[len(state.Signals)-1]
	if last.Kind == SignalContextCompacted {
		decision.Kind = KindAdvise
		decision.Severity = SeverityWarning
		decision.Finding = "Context was compacted and task constraints may be lost"
		decision.Evidence = []string{last.EventID}
		decision.Instruction = "Before continuing, preserve the task constraints, decisions, unresolved work, and current Git state in a bounded handoff."
		decision.ExpiresAt = now.Add(decisionTTL)
		return finalize(decision)
	}

	if evidence, stalled := repeatedInspectionWithoutProgress(state.Signals); stalled {
		decision.Kind = KindAdvise
		decision.Severity = SeverityWarning
		decision.Finding = "Repeated inspection without observable progress"
		decision.Evidence = evidence
		decision.Instruction = "State a concrete hypothesis and take one action that can confirm it or change the project state."
		decision.ProhibitedActions = []string{"Do not inspect the same target again without a new question."}
		decision.ExpiresAt = now.Add(decisionTTL)
		return finalize(decision)
	}

	return finalize(decision)
}

func completionNeedsValidation(signals []Signal) ([]string, bool) {
	lastCompletion := -1
	lastProgress := -1
	for i, signal := range signals {
		if signal.Kind == SignalProgress || signal.Progress {
			lastProgress = i
		}
		if signal.Kind == SignalSessionCompleted {
			lastCompletion = i
		}
	}
	if lastCompletion < 0 {
		return nil, false
	}
	for i := lastProgress + 1; i < lastCompletion; i++ {
		if signals[i].Kind == SignalValidationPassed {
			return nil, false
		}
	}
	return []string{signals[lastCompletion].EventID}, true
}

func trailingRepeatedFailures(signals []Signal) ([]string, bool) {
	if len(signals) < 3 {
		return nil, false
	}
	last := signals[len(signals)-1]
	if last.Kind != SignalToolFailed {
		return nil, false
	}
	evidence := []string{last.EventID}
	for i := len(signals) - 2; i >= 0 && len(evidence) < 3; i-- {
		signal := signals[i]
		if signal.Kind != SignalToolFailed || signal.Tool != last.Tool || signal.InputFingerprint != last.InputFingerprint || signal.ResultFingerprint != last.ResultFingerprint {
			break
		}
		evidence = append(evidence, signal.EventID)
	}
	if len(evidence) != 3 {
		return nil, false
	}
	reverse(evidence)
	return evidence, true
}

func repeatedInspectionWithoutProgress(signals []Signal) ([]string, bool) {
	start := 0
	for i, signal := range signals {
		if signal.Kind == SignalProgress || signal.Progress {
			start = i + 1
		}
	}
	var matched []Signal
	for _, signal := range signals[start:] {
		if signal.Kind != SignalToolCompleted || !isInspectionTool(signal.Tool) {
			matched = nil
			continue
		}
		if len(matched) > 0 {
			previous := matched[len(matched)-1]
			if !strings.EqualFold(previous.Tool, signal.Tool) || previous.InputFingerprint != signal.InputFingerprint {
				matched = nil
			}
		}
		matched = append(matched, signal)
	}
	if len(matched) < 4 {
		return nil, false
	}
	matched = matched[len(matched)-4:]
	evidence := make([]string, 0, len(matched))
	for _, signal := range matched {
		evidence = append(evidence, signal.EventID)
	}
	return evidence, true
}

func isInspectionTool(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "read", "search", "glob", "grep":
		return true
	default:
		return false
	}
}

func finalize(decision Decision) Decision {
	decision.EvidenceFingerprint = hashParts(decision.Evidence...)
	decision.DecisionID = hashParts(decision.SessionID, string(decision.Kind), decision.EvidenceFingerprint)
	return decision
}

func hashParts(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func reverse(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
