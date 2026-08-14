package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestCostIntelligenceAndRequestTrendsUseCapturedRequests(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Now().UTC().Truncate(time.Second)
	for index, status := range []int{200, 500, 200} {
		input, cached, output, reasoning := int64(1000+index*100), int64(800), int64(100), int64(20)
		amount := int64(1000 + index*100)
		completed := now.Add(time.Duration(index) * time.Second)
		cost := conversations.Cost{Currency: "USD", Precision: "exact", Provenance: "catalog"}
		if index == 2 {
			cost.Precision = "unavailable"
			cost.Provenance = "no-catalog"
		} else {
			cost.AmountMicros = &amount
		}
		record := conversations.Request{ID: fmt.Sprintf("insight-%d", index), SessionID: fmt.Sprintf("s-%d", index), ProjectID: "project-insights", Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "gpt-test"}, Protocol: "openai", Method: "POST", Path: "/v1/responses", StatusCode: status, StartedAt: now.Add(time.Duration(index) * time.Second), CompletedAt: &completed, DurationMS: int64((index + 1) * 1000), Usage: conversations.Usage{InputTokens: &input, CachedTokens: &cached, OutputTokens: &output, ReasoningTokens: &reasoning, Precision: "exact", Provenance: "provider"}, Cost: cost}
		if err := database.SaveConversationRequest(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	costs, err := database.CostIntelligence(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if costs.Usage.UncachedInputTokens != costs.Usage.InputTokens-costs.Usage.CachedTokens || costs.Cost.UnknownRequests != 1 || costs.Cost.Availability != "partial" || len(costs.Rankings) == 0 {
		t.Fatalf("costs=%+v", costs)
	}
	trends, err := database.RequestTrends(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(trends.Points) != 1 || trends.Points[0].Requests != 3 || trends.Points[0].Failed != 1 || trends.Points[0].P95LatencyMS < trends.Points[0].P50LatencyMS {
		t.Fatalf("trends=%+v", trends)
	}
}
