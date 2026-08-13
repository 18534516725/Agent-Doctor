// Package comparison provides conservative, matched-cohort comparisons.
package comparison

import (
	"fmt"
	"sort"
)

const minimumWinnerSamples = 15

type Sample struct {
	ProjectID        string
	TaskType         string
	Subject          string
	DurationMS       int64
	ValidationPassed bool
	Corrections      int
	CostMicros       int64
	CostPrecision    string
}

type Request struct {
	ProjectID string
	TaskType  string
	Samples   []Sample
}

type Metric struct {
	Subject                string `json:"subject"`
	Samples                int    `json:"samples"`
	ValidationPassed       int    `json:"validationPassed"`
	MedianDurationMS       int64  `json:"medianDurationMs"`
	Corrections            int    `json:"corrections"`
	ExactCostMicros        int64  `json:"exactCostMicros"`
	ExactCostSamples       int    `json:"exactCostSamples"`
	EstimatedCostMicros    int64  `json:"estimatedCostMicros"`
	EstimatedCostSamples   int    `json:"estimatedCostSamples"`
	UnavailableCostSamples int    `json:"unavailableCostSamples"`
}

type Result struct {
	Cohort         string         `json:"cohort"`
	Samples        map[string]int `json:"samples"`
	Metrics        []Metric       `json:"metrics"`
	Confidence     string         `json:"confidence"`
	Winner         string         `json:"winner,omitempty"`
	Limitations    []string       `json:"limitations"`
	Recommendation string         `json:"recommendation"`
}

func Compare(request Request) (Result, error) {
	if request.ProjectID == "" || request.TaskType == "" {
		return Result{}, fmt.Errorf("project and task type are required for a matched comparison")
	}
	grouped := map[string][]Sample{}
	for _, sample := range request.Samples {
		if sample.ProjectID == request.ProjectID && sample.TaskType == request.TaskType && sample.Subject != "" {
			grouped[sample.Subject] = append(grouped[sample.Subject], sample)
		}
	}
	result := Result{
		Cohort: request.ProjectID + ":" + request.TaskType, Samples: map[string]int{}, Confidence: "low",
		Limitations:    []string{"Observational comparisons cannot establish provider-side causality."},
		Recommendation: "Insufficient balanced evidence to name a winner.",
	}
	subjects := make([]string, 0, len(grouped))
	minSamples, maxSamples := 0, 0
	for subject := range grouped {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	for _, subject := range subjects {
		samples := grouped[subject]
		result.Samples[subject] = len(samples)
		if minSamples == 0 || len(samples) < minSamples {
			minSamples = len(samples)
		}
		if len(samples) > maxSamples {
			maxSamples = len(samples)
		}
		result.Metrics = append(result.Metrics, summarize(subject, samples))
	}
	balanced := len(subjects) >= 2 && minSamples >= minimumWinnerSamples && maxSamples <= minSamples*2
	if balanced {
		result.Confidence = "medium"
		result.Recommendation = "Review the matched metrics; no automatic winner is assigned because duration, validation and cost have different trade-offs."
	} else {
		result.Limitations = append(result.Limitations, "At least fifteen reasonably balanced samples per subject are required before stronger conclusions.")
	}
	return result, nil
}

func summarize(subject string, samples []Sample) Metric {
	durations := make([]int64, 0, len(samples))
	metric := Metric{Subject: subject, Samples: len(samples)}
	for _, sample := range samples {
		if sample.DurationMS >= 0 {
			durations = append(durations, sample.DurationMS)
		}
		if sample.ValidationPassed {
			metric.ValidationPassed++
		}
		metric.Corrections += sample.Corrections
		switch sample.CostPrecision {
		case "exact":
			metric.ExactCostMicros += sample.CostMicros
			metric.ExactCostSamples++
		case "estimated":
			metric.EstimatedCostMicros += sample.CostMicros
			metric.EstimatedCostSamples++
		default:
			metric.UnavailableCostSamples++
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if len(durations) > 0 {
		metric.MedianDurationMS = durations[(len(durations)-1)/2]
	}
	return metric
}
