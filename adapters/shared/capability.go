package shared

import (
	"fmt"
	"time"
)

// CapabilitySet is intentionally conservative: false means unsupported or not
// verified. Product copy must never claim more than its fixture establishes.
type CapabilitySet struct {
	Lifecycle bool
	Context   bool
	MCP       bool
	Model     bool
	Token     bool
	Cost      bool
	Replay    bool
}

type CapabilityFixture struct {
	Client         string
	Version        string
	LastVerifiedAt string
	Capabilities   CapabilitySet
}

func ValidateClaim(fixture CapabilityFixture, claim CapabilitySet) error {
	if fixture.Client == "" || fixture.Version == "" {
		return fmt.Errorf("capability fixture identity is required")
	}
	if _, err := time.Parse("2006-01-02", fixture.LastVerifiedAt); err != nil {
		return fmt.Errorf("capability fixture verification date is invalid")
	}
	if (claim.Lifecycle && !fixture.Capabilities.Lifecycle) ||
		(claim.Context && !fixture.Capabilities.Context) ||
		(claim.MCP && !fixture.Capabilities.MCP) ||
		(claim.Model && !fixture.Capabilities.Model) ||
		(claim.Token && !fixture.Capabilities.Token) ||
		(claim.Cost && !fixture.Capabilities.Cost) ||
		(claim.Replay && !fixture.Capabilities.Replay) {
		return fmt.Errorf("capability claim exceeds verified fixture")
	}
	return nil
}
