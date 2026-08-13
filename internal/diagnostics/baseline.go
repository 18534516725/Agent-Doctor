package diagnostics

import "sort"

type Cohort struct {
	ProjectID  string
	TaskType   string
	Client     string
	ModelMajor string
}

type TaskSample struct {
	ProjectID       string
	TaskType        string
	Client          string
	ModelMajor      string
	DurationSeconds int64
}

type Baseline struct {
	Cohort                Cohort
	SampleCount           int
	Confidence            string
	MedianDurationSeconds *int64
	P90DurationSeconds    *int64
}

// BuildBaseline only compares matching project, task type, client and major
// model version. This avoids announcing a winner from mixed cohorts.
func BuildBaseline(samples []TaskSample, cohort Cohort) Baseline {
	values := make([]int64, 0, len(samples))
	for _, sample := range samples {
		if sample.ProjectID == cohort.ProjectID && sample.TaskType == cohort.TaskType && sample.Client == cohort.Client && sample.ModelMajor == cohort.ModelMajor && sample.DurationSeconds >= 0 {
			values = append(values, sample.DurationSeconds)
		}
	}
	result := Baseline{Cohort: cohort, SampleCount: len(values), Confidence: "facts-only"}
	if len(values) < 5 {
		return result
	}
	sort.Slice(values, func(left, right int) bool { return values[left] < values[right] })
	median := median(values)
	result.MedianDurationSeconds = &median
	if len(values) < 15 {
		result.Confidence = "low"
		return result
	}
	p90 := values[(len(values)*90+99)/100-1]
	result.P90DurationSeconds = &p90
	result.Confidence = "medium"
	return result
}

func median(values []int64) int64 {
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}
