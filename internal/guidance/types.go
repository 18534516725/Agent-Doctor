package guidance

import "time"

type Kind string

const (
	KindContinue Kind = "continue"
	KindAdvise   Kind = "advise"
	KindRedirect Kind = "redirect"
	KindAsk      Kind = "ask"
	KindBlock    Kind = "block"
	KindVerify   Kind = "verify"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type ControlLevel string

const (
	ControlObserve   ControlLevel = "observe"
	ControlGuide     ControlLevel = "guide"
	ControlGuard     ControlLevel = "guard"
	ControlAutopilot ControlLevel = "autopilot"
)

type Decision struct {
	DecisionID          string    `json:"decisionId"`
	SessionID           string    `json:"sessionId"`
	ProjectID           string    `json:"projectId"`
	Kind                Kind      `json:"kind"`
	Severity            Severity  `json:"severity"`
	Finding             string    `json:"finding"`
	Evidence            []string  `json:"evidence"`
	Confidence          string    `json:"confidence"`
	Instruction         string    `json:"instruction"`
	ProhibitedActions   []string  `json:"prohibitedActions"`
	Verification        []string  `json:"verification"`
	EvidenceFingerprint string    `json:"evidenceFingerprint"`
	ExpiresAt           time.Time `json:"expiresAt"`
	CreatedAt           time.Time `json:"createdAt"`
}

// DeliveryReceipt proves that a client fetched one guidance decision. It is
// deliberately content-free and never contains prompts, code, commands, paths,
// tool results, or credentials.
type DeliveryReceipt struct {
	SessionID     string       `json:"sessionId"`
	ProjectID     string       `json:"projectId"`
	Client        string       `json:"client"`
	DecisionID    string       `json:"decisionId"`
	DecisionKind  Kind         `json:"decisionKind"`
	ControlLevel  ControlLevel `json:"controlLevel"`
	DeliveryCount int          `json:"deliveryCount"`
	DeliveredAt   time.Time    `json:"deliveredAt"`
}

type SignalKind string

const (
	SignalToolFailed       SignalKind = "tool.failed"
	SignalToolCompleted    SignalKind = "tool.completed"
	SignalProgress         SignalKind = "progress"
	SignalContextCompacted SignalKind = "context.compacted"
	SignalSessionCompleted SignalKind = "session.completed"
	SignalValidationPassed SignalKind = "validation.passed"
)

// Signal is deliberately content-free. It carries only bounded labels,
// progress facts, source IDs and non-reversible fingerprints.
type Signal struct {
	EventID           string
	Kind              SignalKind
	Tool              string
	InputFingerprint  string
	ResultFingerprint string
	Progress          bool
	At                time.Time
}

type SessionState struct {
	SessionID string
	ProjectID string
	Signals   []Signal
}
