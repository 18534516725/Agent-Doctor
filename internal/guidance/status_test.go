package guidance

import (
	"errors"
	"testing"
	"time"
)

func TestGuidanceStatusReflectsEvidenceAndCapability(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Minute)
	stale := now.Add(-20 * time.Minute)
	tests := []struct {
		name       string
		last       *time.Time
		client     string
		active     bool
		repository error
		want       ConnectionState
		advice     bool
		enforce    bool
	}{
		{name: "no evidence", want: StateUnavailable},
		{name: "codex observing", last: &recent, client: "codex", want: StateObserving, advice: true},
		{name: "codex active", last: &recent, client: "codex", active: true, want: StateActive, advice: true},
		{name: "claude active", last: &recent, client: "claude-code", active: true, want: StateActive, advice: true, enforce: true},
		{name: "stale", last: &stale, client: "codex", want: StateStale, advice: true},
		{name: "repository error", repository: errors.New("sqlite unavailable"), want: StateError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveStatus(test.last, test.client, test.active, now, test.repository)
			if got.State != test.want || got.Advice != test.advice || got.Enforcement != test.enforce {
				t.Fatalf("status=%+v", got)
			}
		})
	}
}
