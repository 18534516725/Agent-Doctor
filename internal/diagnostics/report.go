package diagnostics

type TaskFacts struct {
	ProjectID                        string
	Client                           string
	ModelMajor                       string
	CompactionCount                  int
	MissingConstraintAfterCompaction bool
	ValidationFailures               int
	ComparableSamples                int
	RepeatedDegradation              bool
}

type Finding struct {
	Cause           string
	Confidence      string
	Evidence        []string
	CounterEvidence []string
}

type DiagnosticReport struct {
	Primary     Finding
	Limitations []string
	NextAction  string
}

// Diagnose maps known facts to a bounded recommendation. It never asserts a
// model regression without enough matched evidence across the same cohort.
func Diagnose(facts TaskFacts) DiagnosticReport {
	report := DiagnosticReport{Limitations: []string{"No prompt, source content, or provider-internal data was used."}}
	if facts.CompactionCount > 0 && facts.MissingConstraintAfterCompaction {
		report.Primary = Finding{
			Cause: "context-compaction-followed-by-missed-constraint", Confidence: "medium",
			Evidence:        []string{"A context compaction occurred before a required constraint was missed."},
			CounterEvidence: []string{"This does not prove the compaction was the only cause."},
		}
		report.NextAction = "Save the missing constraint as explicit project memory and retry with a compact task capsule."
		return report
	}
	if facts.ValidationFailures > 0 {
		report.Primary = Finding{
			Cause: "validation-environment-failure", Confidence: "medium",
			Evidence:        []string{"An approved validation command failed."},
			CounterEvidence: []string{"The task implementation may still contain an independent issue."},
		}
		report.NextAction = "Inspect the validation environment and rerun the approved command before judging the model output."
		return report
	}
	if facts.ComparableSamples >= 15 && facts.RepeatedDegradation {
		report.Primary = Finding{
			Cause: "suspected-model-anomaly", Confidence: "low",
			Evidence:        []string{"Matched project, task type, client, and model-major samples show repeated degradation."},
			CounterEvidence: []string{"Observational data cannot establish a provider-side cause."},
		}
		report.NextAction = "Run a consented, capped replay against the matched baseline before changing models."
		return report
	}
	report.Primary = Finding{
		Cause: "insufficient-evidence", Confidence: "unavailable",
		Evidence:        []string{"No deterministic anomaly rule had enough supporting facts."},
		CounterEvidence: []string{"Absence of evidence is not evidence that the task was normal."},
	}
	report.Limitations = append(report.Limitations, "At least fifteen matched samples are required before a model-anomaly finding can lead.")
	report.NextAction = "Collect more approved local task facts or inspect the task timeline manually."
	return report
}
