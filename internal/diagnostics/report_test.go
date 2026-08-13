package diagnostics

import "testing"

func TestDiagnoseProducesEvidenceCounterEvidenceLimitsAndAction(t *testing.T) {
	report := Diagnose(TaskFacts{
		CompactionCount: 1, MissingConstraintAfterCompaction: true, ValidationFailures: 0,
		ComparableSamples: 16, RepeatedDegradation: false, ProjectID: "project-a", Client: "Codex", ModelMajor: "5",
	})
	if report.Primary.Cause != "context-compaction-followed-by-missed-constraint" || report.Primary.Confidence == "unavailable" || len(report.Primary.Evidence) == 0 || len(report.Primary.CounterEvidence) == 0 || report.Limitations == nil || report.NextAction == "" {
		t.Fatalf("report=%+v", report)
	}
}

func TestDiagnoseDoesNotLeadWithModelAnomalyWithoutMatchedRepeatedEvidence(t *testing.T) {
	report := Diagnose(TaskFacts{ComparableSamples: 4, RepeatedDegradation: true, ProjectID: "project-a", Client: "Codex", ModelMajor: "5"})
	if report.Primary.Cause == "suspected-model-anomaly" || report.Primary.Confidence != "unavailable" {
		t.Fatalf("report=%+v", report)
	}
}

func TestDiagnoseHandlesInsufficientEvidenceHonestly(t *testing.T) {
	report := Diagnose(TaskFacts{})
	if report.Primary.Cause != "insufficient-evidence" || report.NextAction == "" || len(report.Limitations) == 0 {
		t.Fatalf("report=%+v", report)
	}
}
