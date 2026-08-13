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
