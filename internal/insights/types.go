package insights

type Usage struct {
	InputTokens         int64   `json:"inputTokens"`
	CachedTokens        int64   `json:"cachedTokens"`
	UncachedInputTokens int64   `json:"uncachedInputTokens"`
	OutputTokens        int64   `json:"outputTokens"`
	ReasoningTokens     int64   `json:"reasoningTokens"`
	CacheRate           float64 `json:"cacheRate"`
}

type Cost struct {
	Currency        string `json:"currency"`
	ExactMicros     int64  `json:"exactMicros"`
	EstimatedMicros int64  `json:"estimatedMicros"`
	UnknownRequests int    `json:"unknownRequests"`
	Availability    string `json:"availability"`
}

type Ranking struct {
	Dimension           string `json:"dimension"`
	Label               string `json:"label"`
	Requests            int    `json:"requests"`
	UncachedInputTokens int64  `json:"uncachedInputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	ExactMicros         int64  `json:"exactMicros"`
	UnknownCosts        int    `json:"unknownCosts"`
}

type UnknownCost struct {
	RequestID  string `json:"requestId"`
	Model      string `json:"model"`
	Client     string `json:"client"`
	StartedAt  string `json:"startedAt"`
	Provenance string `json:"provenance"`
}
type CostIntelligence struct {
	Days        int           `json:"days"`
	Usage       Usage         `json:"usage"`
	Cost        Cost          `json:"cost"`
	Rankings    []Ranking     `json:"rankings"`
	Unknown     []UnknownCost `json:"unknown"`
	Limitations []string      `json:"limitations"`
}

type TrendPoint struct {
	Date                string  `json:"date"`
	Requests            int     `json:"requests"`
	Failed              int     `json:"failed"`
	FailureRate         float64 `json:"failureRate"`
	P50LatencyMS        int64   `json:"p50LatencyMs"`
	P95LatencyMS        int64   `json:"p95LatencyMs"`
	UncachedInputTokens int64   `json:"uncachedInputTokens"`
	OutputTokens        int64   `json:"outputTokens"`
	CachedTokens        int64   `json:"cachedTokens"`
	CacheRate           float64 `json:"cacheRate"`
}
type RequestTrends struct {
	Days        int          `json:"days"`
	Points      []TrendPoint `json:"points"`
	Limitations []string     `json:"limitations"`
}
