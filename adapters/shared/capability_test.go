package shared

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCapabilityClaimsNeverExceedVerifiedFixture(t *testing.T) {
	fixture := CapabilityFixture{
		Client: "Continue", Version: "1.0", LastVerifiedAt: "2026-08-13",
		Capabilities: CapabilitySet{MCP: true, Context: true, Lifecycle: false, Token: false, Cost: false, Replay: false},
	}
	claim := CapabilitySet{MCP: true, Context: true}
	if err := ValidateClaim(fixture, claim); err != nil {
		t.Fatal(err)
	}
	if err := ValidateClaim(fixture, CapabilitySet{MCP: true, Lifecycle: true}); err == nil {
		t.Fatal("expected unsupported lifecycle claim to fail")
	}
}

func TestCapabilityFixtureRequiresIdentityAndVerificationDate(t *testing.T) {
	if err := ValidateClaim(CapabilityFixture{}, CapabilitySet{}); err == nil {
		t.Fatal("expected invalid fixture to fail")
	}
}

func TestPublishedCapabilityFixturesAreValid(t *testing.T) {
	raw, err := os.ReadFile("capabilities.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []CapabilityFixture
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 3 {
		t.Fatalf("fixtures=%+v", fixtures)
	}
	for _, fixture := range fixtures {
		if err := ValidateClaim(fixture, fixture.Capabilities); err != nil {
			t.Fatalf("fixture=%+v err=%v", fixture, err)
		}
	}
}
