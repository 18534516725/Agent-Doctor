package context

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestCapsuleRanksMandatoryCurrentScopeAndCollapsesDuplicates(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	request := CapsuleRequest{
		ProjectID:   "project-1",
		CurrentFile: "internal/app.go",
		Budget:      800,
		Now:         now,
		Candidates: []CapsuleCandidate{
			{Memory: activeMemory("rule", "project-1", "Never expose credentials", SourceRepositoryRule, 0.98), Scope: ScopeProject, Mandatory: true},
			{Memory: activeMemory("file", "project-1", "Use the existing app constructor", SourceInferred, 0.8), Scope: ScopeFile, FilePath: "internal/app.go"},
			{Memory: activeMemory("preference", "project-1", "Prefer concise output", SourceUserExplicit, 1), Scope: ScopeProject},
			{Memory: activeMemory("global", "", "Global historical habit", SourceInferred, 0.8), Scope: ScopeGlobal},
			{Memory: activeMemory("duplicate-low", "project-1", "never   expose credentials", SourceInferred, 0.4), Scope: ScopeProject},
		},
	}
	capsule := BuildCapsule(request)
	if capsule.TokenEstimate > request.Budget {
		t.Fatalf("tokens=%d budget=%d", capsule.TokenEstimate, request.Budget)
	}
	if strings.Count(strings.ToLower(capsule.Rendered), "never expose credentials") != 1 {
		t.Fatalf("duplicate was not collapsed: %s", capsule.Rendered)
	}
	ruleIndex := strings.Index(capsule.Rendered, "Never expose credentials")
	fileIndex := strings.Index(capsule.Rendered, "Use the existing app constructor")
	preferenceIndex := strings.Index(capsule.Rendered, "Prefer concise output")
	globalIndex := strings.Index(capsule.Rendered, "Global historical habit")
	if ruleIndex < 0 || fileIndex < 0 || preferenceIndex < 0 || globalIndex < 0 || !(ruleIndex < preferenceIndex && fileIndex < globalIndex) {
		t.Fatalf("ranking incorrect: %s", capsule.Rendered)
	}
	for _, provenance := range []string{"repository-rule", "inferred", "user-explicit"} {
		if !strings.Contains(capsule.Rendered, "source: "+provenance) {
			t.Fatalf("missing provenance %q: %s", provenance, capsule.Rendered)
		}
	}
}

func TestCapsuleExcludesExpiredDisabledAndOtherProjects(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	expired := activeMemory("expired", "project-1", "expired text", SourceUserExplicit, 1)
	expired.ExpiresAt = &expiredAt
	disabled := activeMemory("disabled", "project-1", "disabled text", SourceRepositoryRule, 1)
	disabled.State = MemoryDisabled
	other := activeMemory("other", "project-2", "other project text", SourceRepositoryRule, 1)
	capsule := BuildCapsule(CapsuleRequest{
		ProjectID: "project-1", Budget: 800, Now: now,
		Candidates: []CapsuleCandidate{{Memory: expired}, {Memory: disabled}, {Memory: other}},
	})
	for _, forbidden := range []string{"expired text", "disabled text", "other project text"} {
		if strings.Contains(capsule.Rendered, forbidden) {
			t.Fatalf("ineligible memory included: %s", capsule.Rendered)
		}
	}
}

func TestCapsuleShowsUnavailableContextAndHonorsDefaultBudget(t *testing.T) {
	candidates := make([]CapsuleCandidate, 0, 100)
	for index := 0; index < 100; index++ {
		candidates = append(candidates, CapsuleCandidate{Memory: activeMemory(
			string(rune('a'+index%26))+strings.Repeat("x", index/26), "project-1",
			strings.Repeat("context detail ", 20), SourceInferred, 0.7,
		)})
	}
	capsule := BuildCapsule(CapsuleRequest{
		ProjectID:   "project-1",
		Candidates:  candidates,
		Unavailable: []UnavailableContext{{Label: "exact quota", Reason: "client does not expose quota telemetry"}},
	})
	if capsule.Budget != DefaultCapsuleBudget || capsule.TokenEstimate > DefaultCapsuleBudget {
		t.Fatalf("capsule=%+v", capsule)
	}
	if !strings.Contains(capsule.Rendered, "Unavailable context") || !strings.Contains(capsule.Rendered, "exact quota") {
		t.Fatalf("unavailable context missing: %s", capsule.Rendered)
	}
}

func FuzzCapsuleNeverExceedsBudgetOrIncludesDisabledText(f *testing.F) {
	f.Add("ordinary memory", "disabled-marker", 80)
	f.Add("中文上下文规则", "不要出现的内容", 32)
	f.Fuzz(func(t *testing.T, activeText, disabledText string, budget int) {
		if budget < 1 {
			budget = 1
		}
		if budget > 2000 {
			budget = 2000
		}
		disabledMarker := "disabled-memory-marker-" + base64.RawURLEncoding.EncodeToString([]byte(disabledText))
		disabled := activeMemory("disabled", "project-1", disabledMarker, SourceRepositoryRule, 1)
		disabled.State = MemoryDisabled
		capsule := BuildCapsule(CapsuleRequest{
			ProjectID: "project-1", Budget: budget,
			Candidates: []CapsuleCandidate{
				{Memory: activeMemory("active", "project-1", activeText, SourceUserExplicit, 1)},
				{Memory: disabled},
			},
		})
		if capsule.TokenEstimate > budget {
			t.Fatalf("tokens=%d budget=%d", capsule.TokenEstimate, budget)
		}
		if !strings.Contains(activeText, disabledMarker) && strings.Contains(capsule.Rendered, disabledMarker) {
			t.Fatalf("disabled content included")
		}
	})
}

func activeMemory(id, projectID, text string, source MemorySource, confidence float64) Memory {
	return Memory{
		ID: id, ProjectID: projectID, Text: text, Source: source, Confidence: confidence,
		ObservationCount: 3, State: MemoryActive,
	}
}
