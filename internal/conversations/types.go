package conversations

import (
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type Usage struct {
	InputTokens     *int64 `json:"inputTokens,omitempty"`
	OutputTokens    *int64 `json:"outputTokens,omitempty"`
	CachedTokens    *int64 `json:"cachedTokens,omitempty"`
	ReasoningTokens *int64 `json:"reasoningTokens,omitempty"`
	Precision       string `json:"precision"`
	Provenance      string `json:"provenance"`
}

type Cost struct {
	AmountMicros *int64 `json:"amountMicros,omitempty"`
	Currency     string `json:"currency"`
	Precision    string `json:"precision"`
	Provenance   string `json:"provenance"`
}

type Message struct {
	ID              string    `json:"id"`
	RequestID       string    `json:"requestId"`
	Sequence        int       `json:"sequence"`
	Role            string    `json:"role"`
	Content         string    `json:"content"`
	ToolName        string    `json:"toolName,omitempty"`
	ToolPayloadJSON string    `json:"toolPayloadJson,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Request struct {
	ID          string           `json:"id"`
	SessionID   string           `json:"sessionId"`
	ProjectID   string           `json:"projectId"`
	Client      events.ClientRef `json:"client"`
	Model       events.ModelRef  `json:"model"`
	Protocol    string           `json:"protocol"`
	Method      string           `json:"method"`
	Path        string           `json:"path"`
	StatusCode  int              `json:"statusCode"`
	StartedAt   time.Time        `json:"startedAt"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	FirstByteMS int64            `json:"firstByteMs"`
	DurationMS  int64            `json:"durationMs"`
	Usage       Usage            `json:"usage"`
	Cost        Cost             `json:"cost"`
	Messages    []Message        `json:"messages"`
}

type ClientConnection struct {
	Key             string     `json:"key"`
	DisplayName     string     `json:"displayName"`
	Detected        bool       `json:"detected"`
	State           string     `json:"state"`
	Capability      string     `json:"capability"`
	Detail          string     `json:"detail"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type LiveAnalysis struct {
	ProjectID           string    `json:"projectId"`
	Requests            int       `json:"requests"`
	ActiveSessions      int       `json:"activeSessions"`
	InputTokens         int64     `json:"inputTokens"`
	OutputTokens        int64     `json:"outputTokens"`
	CachedTokens        int64     `json:"cachedTokens"`
	ReasoningTokens     int64     `json:"reasoningTokens"`
	ExactCostMicros     int64     `json:"exactCostMicros"`
	EstimatedCostMicros int64     `json:"estimatedCostMicros"`
	UnknownCostCount    int       `json:"unknownCostCount"`
	AverageLatencyMS    float64   `json:"averageLatencyMs"`
	FailedRequests      int       `json:"failedRequests"`
	ToolCalls           int       `json:"toolCalls"`
	TokensPerRequest    float64   `json:"tokensPerRequest"`
	CacheHitRate        float64   `json:"cacheHitRate"`
	HealthScore         int       `json:"healthScore"`
	Summary             string    `json:"summary"`
	Findings            []Finding `json:"findings"`
	Limitations         []string  `json:"limitations"`
}

type Finding struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}

type PrivacySettings struct {
	CapturePrompts      bool `json:"capturePrompts"`
	CaptureFileContents bool `json:"captureFileContents"`
	RetentionDays       int  `json:"retentionDays"`
}
