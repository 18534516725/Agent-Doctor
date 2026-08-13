// Package dashboard defines safe, aggregate-only views for the local UI.
package dashboard

import "context"

// PrecisionCounts reports evidence quality without exposing event payloads.
type PrecisionCounts struct {
	Exact       int `json:"exact"`
	Estimated   int `json:"estimated"`
	Unavailable int `json:"unavailable"`
}

// Summary intentionally contains counts only. Raw prompts, file contents,
// commands, credentials and provider internals never cross this boundary.
type Summary struct {
	Projects       int             `json:"projects"`
	Sessions       int             `json:"sessions"`
	ActiveSessions int             `json:"activeSessions"`
	Events         int             `json:"events"`
	Precision      PrecisionCounts `json:"precision"`
}

// SummaryProvider is implemented by local storage when aggregation is ready.
type SummaryProvider interface {
	DashboardSummary(context.Context) (Summary, error)
}

type Session struct {
	ID         string `json:"id"`
	Client     string `json:"client"`
	Model      string `json:"model"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	EventCount int    `json:"eventCount"`
}

type Costs struct {
	Currency        string `json:"currency"`
	ExactMicros     int64  `json:"exactMicros"`
	EstimatedMicros int64  `json:"estimatedMicros"`
	Unavailable     int    `json:"unavailable"`
}

type Memories struct {
	Active    int `json:"active"`
	Candidate int `json:"candidate"`
	Disabled  int `json:"disabled"`
}

type TrendPoint struct {
	Date     string `json:"date"`
	Sessions int    `json:"sessions"`
	Events   int    `json:"events"`
}

type Snapshot struct {
	Sessions        []Session    `json:"sessions"`
	Costs           Costs        `json:"costs"`
	Memories        Memories     `json:"memories"`
	Trends          []TrendPoint `json:"trends"`
	ComparisonCount int          `json:"comparisonCount"`
}

type SnapshotProvider interface {
	DashboardSnapshot(context.Context) (Snapshot, error)
}
