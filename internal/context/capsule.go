package context

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

const DefaultCapsuleBudget = 800

type UnavailableContext struct {
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

type CapsuleRequest struct {
	ProjectID   string
	CurrentFile string
	Candidates  []CapsuleCandidate
	Unavailable []UnavailableContext
	Budget      int
	Now         time.Time
}

type Capsule struct {
	Rendered      string             `json:"rendered"`
	TokenEstimate int                `json:"tokenEstimate"`
	Budget        int                `json:"budget"`
	Selected      []CapsuleCandidate `json:"selected"`
}

type capsuleSection struct {
	title      string
	candidates []CapsuleCandidate
	lines      []string
}

func BuildCapsule(request CapsuleRequest) Capsule {
	if request.Budget <= 0 {
		request.Budget = DefaultCapsuleBudget
	}
	if request.Now.IsZero() {
		request.Now = time.Now().UTC()
	}
	ranked := rankCandidates(request)
	sections := []capsuleSection{{title: "Must follow"}, {title: "User preferences"}, {title: "Relevant history"}}
	for _, candidate := range ranked {
		index := 2
		if candidate.Mandatory || candidate.Memory.Source == SourceRepositoryRule {
			index = 0
		} else if candidate.Memory.Source == SourceUserExplicit {
			index = 1
		}
		sections[index].candidates = append(sections[index].candidates, candidate)
	}

	unavailable := buildUnavailableSection(request.Unavailable, request.Budget)
	reserved := 0
	if unavailable != "" {
		reserved = EstimateTokens(unavailable)
	}
	availableBudget := request.Budget - reserved
	if unavailable != "" {
		availableBudget -= EstimateTokens("\n\n")
	}
	if availableBudget < 0 {
		availableBudget = 0
	}

	selected := make([]CapsuleCandidate, 0)
	parts := make([]string, 0, 4)
	for _, section := range sections {
		sectionText, sectionSelected := renderCandidateSection(section, availableBudget-EstimateTokens(strings.Join(parts, "\n\n")))
		if sectionText == "" {
			continue
		}
		candidateParts := append(append([]string(nil), parts...), sectionText)
		if EstimateTokens(strings.Join(candidateParts, "\n\n")) > availableBudget {
			continue
		}
		parts = candidateParts
		selected = append(selected, sectionSelected...)
	}
	if unavailable != "" {
		candidateParts := append(append([]string(nil), parts...), unavailable)
		if EstimateTokens(strings.Join(candidateParts, "\n\n")) <= request.Budget {
			parts = candidateParts
		}
	}
	rendered := strings.Join(parts, "\n\n")
	return Capsule{Rendered: rendered, TokenEstimate: EstimateTokens(rendered), Budget: request.Budget, Selected: selected}
}

func renderCandidateSection(section capsuleSection, budget int) (string, []CapsuleCandidate) {
	if budget <= 0 || len(section.candidates) == 0 {
		return "", nil
	}
	lines := make([]string, 0, len(section.candidates))
	selected := make([]CapsuleCandidate, 0, len(section.candidates))
	for _, candidate := range section.candidates {
		text := strings.TrimSpace(privacy.FilterText(candidate.Memory.Text))
		if text == "" {
			continue
		}
		line := fmt.Sprintf("- %s (source: %s; scope: %s)", text, candidate.Memory.Source, scopeLabel(candidate))
		candidateText := "## " + section.title + "\n" + strings.Join(append(append([]string(nil), lines...), line), "\n")
		if EstimateTokens(candidateText) > budget {
			continue
		}
		lines = append(lines, line)
		selected = append(selected, candidate)
	}
	if len(lines) == 0 {
		return "", nil
	}
	return "## " + section.title + "\n" + strings.Join(lines, "\n"), selected
}

func buildUnavailableSection(unavailable []UnavailableContext, budget int) string {
	lines := make([]string, 0, len(unavailable))
	for _, item := range unavailable {
		label := strings.TrimSpace(privacy.FilterText(item.Label))
		reason := strings.TrimSpace(privacy.FilterText(item.Reason))
		if label == "" || reason == "" {
			continue
		}
		line := fmt.Sprintf("- %s: %s", label, reason)
		candidate := "## Unavailable context\n" + strings.Join(append(append([]string(nil), lines...), line), "\n")
		if EstimateTokens(candidate) > budget {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return "## Unavailable context\n" + strings.Join(lines, "\n")
}

func scopeLabel(candidate CapsuleCandidate) string {
	if candidate.Scope == ScopeFile && candidate.FilePath != "" {
		return "file:" + candidate.FilePath
	}
	if candidate.Scope == "" {
		return string(ScopeProject)
	}
	return string(candidate.Scope)
}

// EstimateTokens is intentionally conservative and language-independent:
// every non-ASCII rune counts as one token and every three ASCII runes as one.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	ascii := 0
	nonASCII := 0
	for len(text) > 0 {
		runeValue, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if runeValue <= 0x7f {
			ascii++
		} else {
			nonASCII++
		}
	}
	return (ascii+2)/3 + nonASCII
}
