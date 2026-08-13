package diagnostics

import "testing"

func TestBuildBaselineUsesFactsOnlyBelowFiveComparableSamples(t *testing.T) {
	baseline := BuildBaseline([]TaskSample{
		{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: 10},
		{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: 20},
		{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: 30},
		{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: 40},
	}, Cohort{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5"})
	if baseline.SampleCount != 4 || baseline.Confidence != "facts-only" || baseline.MedianDurationSeconds != nil {
		t.Fatalf("baseline=%+v", baseline)
	}
}

func TestBuildBaselineUsesLowConfidenceTrendForFiveToFourteenSamples(t *testing.T) {
	samples := make([]TaskSample, 0, 5)
	for _, duration := range []int64{10, 20, 30, 40, 50} {
		samples = append(samples, TaskSample{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: duration})
	}
	baseline := BuildBaseline(samples, Cohort{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5"})
	if baseline.SampleCount != 5 || baseline.Confidence != "low" || baseline.MedianDurationSeconds == nil || *baseline.MedianDurationSeconds != 30 || baseline.P90DurationSeconds != nil {
		t.Fatalf("baseline=%+v", baseline)
	}
}

func TestBuildBaselineSplitsProjectClientAndModelMajor(t *testing.T) {
	samples := make([]TaskSample, 0, 18)
	for index := 0; index < 15; index++ {
		samples = append(samples, TaskSample{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: int64(index + 1)})
	}
	samples = append(samples,
		TaskSample{ProjectID: "b", TaskType: "bugfix", Client: "Codex", ModelMajor: "5", DurationSeconds: 999},
		TaskSample{ProjectID: "a", TaskType: "bugfix", Client: "Claude Code", ModelMajor: "5", DurationSeconds: 999},
		TaskSample{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "6", DurationSeconds: 999},
	)
	baseline := BuildBaseline(samples, Cohort{ProjectID: "a", TaskType: "bugfix", Client: "Codex", ModelMajor: "5"})
	if baseline.SampleCount != 15 || baseline.Confidence != "medium" || baseline.MedianDurationSeconds == nil || *baseline.MedianDurationSeconds != 8 || baseline.P90DurationSeconds == nil || *baseline.P90DurationSeconds != 14 {
		t.Fatalf("baseline=%+v", baseline)
	}
}
