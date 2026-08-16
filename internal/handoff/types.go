package handoff

import "time"

const DefaultBudget = 800

type Memory struct {
	Content    string `json:"content"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId,omitempty"`
}

type Snapshot struct {
	ProjectID       string    `json:"projectId"`
	SourceClient    string    `json:"sourceClient"`
	SourceSessionID string    `json:"sourceSessionId"`
	Goal            string    `json:"goal"`
	LatestResult    string    `json:"latestResult"`
	Memories        []Memory  `json:"memories"`
	GeneratedAt     time.Time `json:"generatedAt"`
	Limitations     []string  `json:"limitations"`
}

type Delivery struct {
	ProjectID       string    `json:"projectId"`
	SourceClient    string    `json:"sourceClient"`
	SourceSessionID string    `json:"sourceSessionId"`
	TargetClient    string    `json:"targetClient"`
	MemoryCount     int       `json:"memoryCount"`
	DeliveredAt     time.Time `json:"deliveredAt"`
}

type Capsule struct {
	Snapshot
	Rendered      string    `json:"rendered"`
	TokenEstimate int       `json:"tokenEstimate"`
	Budget        int       `json:"budget"`
	Provenance    string    `json:"provenance"`
	LastDelivery  *Delivery `json:"lastDelivery,omitempty"`
}
