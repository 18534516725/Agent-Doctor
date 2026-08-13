package comparison

import "testing"

func TestCompareUsesOnlyMatchedCohortAndKeepsCostPrecisionSeparate(t *testing.T) {
	samples := []Sample{
		{ProjectID: "p1", TaskType: "bugfix", Subject: "codex", DurationMS: 1000, ValidationPassed: true, CostMicros: 1200, CostPrecision: "exact"},
		{ProjectID: "p1", TaskType: "bugfix", Subject: "codex", DurationMS: 1200, ValidationPassed: true, CostMicros: 1000, CostPrecision: "estimated"},
		{ProjectID: "p1", TaskType: "bugfix", Subject: "claude-code", DurationMS: 1400, ValidationPassed: false, Corrections: 1, CostMicros: 900, CostPrecision: "exact"},
		{ProjectID: "other", TaskType: "bugfix", Subject: "claude-code", DurationMS: 1, ValidationPassed: true, CostMicros: 1, CostPrecision: "exact"},
	}
	result, err := Compare(Request{ProjectID: "p1", TaskType: "bugfix", Samples: samples})
	if err != nil {
		t.Fatal(err)
	}
	if result.Samples["codex"] != 2 || result.Samples["claude-code"] != 1 {
		t.Fatalf("samples=%v", result.Samples)
	}
	if result.Recommendation != "Insufficient balanced evidence to name a winner." || result.Confidence != "low" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Metrics) != 2 || result.Metrics[0].ExactCostMicros == result.Metrics[0].EstimatedCostMicros {
		t.Fatalf("metrics=%+v", result.Metrics)
	}
}

func TestCompareRejectsMissingCohortAndWinnerForSmallSamples(t *testing.T) {
	if _, err := Compare(Request{}); err == nil {
		t.Fatal("missing matched cohort was accepted")
	}
	samples := make([]Sample, 0, 10)
	for index := 0; index < 5; index++ {
		samples = append(samples,
			Sample{ProjectID: "p1", TaskType: "feature", Subject: "a", DurationMS: 100, ValidationPassed: true},
			Sample{ProjectID: "p1", TaskType: "feature", Subject: "b", DurationMS: 200, ValidationPassed: false},
		)
	}
	result, err := Compare(Request{ProjectID: "p1", TaskType: "feature", Samples: samples})
	if err != nil {
		t.Fatal(err)
	}
	if result.Winner != "" || result.Confidence != "low" {
		t.Fatalf("small sample produced winner: %+v", result)
	}
}
