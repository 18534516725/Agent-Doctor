package context

import (
	"errors"
	"testing"
	"time"
)

func TestMemoryExplicitPreferenceIsImmediatelyEligible(t *testing.T) {
	manager := NewMemoryManager(3)
	memory, err := manager.Observe(Observation{
		ID: "formatting", ProjectID: "project-1", Text: "Use tabs", Source: SourceUserExplicit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if memory.State != MemoryActive || memory.Confidence != 1 {
		t.Fatalf("memory=%+v", memory)
	}
	if got := manager.ListEligible("project-1", time.Now()); len(got) != 1 {
		t.Fatalf("eligible=%v", got)
	}
}

func TestMemoryRepositoryRuleOutranksInferredBehavior(t *testing.T) {
	manager := NewMemoryManager(2)
	for index := 0; index < 2; index++ {
		_, err := manager.Observe(Observation{
			ID: "inferred", ProjectID: "project-1", Text: "Usually run npm", Source: SourceInferred,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := manager.Observe(Observation{
		ID: "rule", ProjectID: "project-1", Text: "Always use pnpm", Source: SourceRepositoryRule,
	}); err != nil {
		t.Fatal(err)
	}
	eligible := manager.ListEligible("project-1", time.Now())
	if len(eligible) != 2 || eligible[0].Source != SourceRepositoryRule {
		t.Fatalf("priority=%+v", eligible)
	}
}

func TestMemoryOneInferenceNeverBecomesPersistent(t *testing.T) {
	manager := NewMemoryManager(3)
	memory, err := manager.Observe(Observation{
		ID: "style", ProjectID: "project-1", Text: "Prefers concise output", Source: SourceInferred,
	})
	if err != nil {
		t.Fatal(err)
	}
	if memory.State != MemoryCandidate || len(manager.ListEligible("project-1", time.Now())) != 0 {
		t.Fatalf("one observation became persistent: %+v", memory)
	}
}

func TestMemoryInferenceRequiresConfiguredThreshold(t *testing.T) {
	manager := NewMemoryManager(3)
	for index := 1; index <= 3; index++ {
		memory, err := manager.Observe(Observation{
			ID: "tests", ProjectID: "project-1", Text: "Runs focused tests first", Source: SourceInferred,
		})
		if err != nil {
			t.Fatal(err)
		}
		if index < 3 && memory.State != MemoryCandidate {
			t.Fatalf("observation %d activated early: %+v", index, memory)
		}
		if index == 3 && memory.State != MemoryActive {
			t.Fatalf("threshold did not activate: %+v", memory)
		}
	}
}

func TestMemoryDeletedItemNeverReappearsFromCache(t *testing.T) {
	manager := NewMemoryManager(2)
	if _, err := manager.Observe(Observation{
		ID: "deleted", ProjectID: "project-1", Text: "Old preference", Source: SourceUserExplicit,
	}); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListEligible("project-1", time.Now())) != 1 {
		t.Fatal("failed to prime eligible cache")
	}
	if err := manager.Delete("deleted"); err != nil {
		t.Fatal(err)
	}
	if got := manager.ListEligible("project-1", time.Now()); len(got) != 0 {
		t.Fatalf("deleted memory survived cache: %+v", got)
	}
	_, err := manager.Observe(Observation{
		ID: "deleted", ProjectID: "project-1", Text: "Old preference", Source: SourceInferred,
	})
	if !errors.Is(err, ErrMemoryDeleted) {
		t.Fatalf("deleted tombstone was recreated: %v", err)
	}
}

func TestMemoryExpiryAndDisableAreHonored(t *testing.T) {
	manager := NewMemoryManager(2)
	expired := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	if _, err := manager.Observe(Observation{
		ID: "temporary", ProjectID: "project-1", Text: "Temporary rule", Source: SourceUserExplicit,
		ExpiresAt: &expired,
	}); err != nil {
		t.Fatal(err)
	}
	if got := manager.ListEligible("project-1", expired.Add(time.Hour)); len(got) != 0 {
		t.Fatalf("expired memory eligible: %+v", got)
	}
	if _, err := manager.Observe(Observation{
		ID: "disabled", ProjectID: "project-1", Text: "Disabled rule", Source: SourceRepositoryRule,
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Disable("disabled"); err != nil {
		t.Fatal(err)
	}
	if got := manager.ListEligible("project-1", time.Now()); len(got) != 0 {
		t.Fatalf("disabled memory eligible: %+v", got)
	}
	if _, err := manager.Observe(Observation{
		ID: "disabled", ProjectID: "project-1", Text: "Disabled rule", Source: SourceInferred,
	}); !errors.Is(err, ErrMemoryDisabled) {
		t.Fatalf("disabled memory was silently re-enabled: %v", err)
	}
	if err := manager.Enable("disabled"); err != nil {
		t.Fatal(err)
	}
	if got := manager.ListEligible("project-1", time.Now()); len(got) != 1 {
		t.Fatalf("explicitly enabled memory missing: %+v", got)
	}
}
