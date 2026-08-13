package context

import (
	"path/filepath"
	"sort"
	"strings"
)

type MemoryScope string

const (
	ScopeFile    MemoryScope = "file"
	ScopeProject MemoryScope = "project"
	ScopeGlobal  MemoryScope = "global"
)

type CapsuleCandidate struct {
	Memory    Memory      `json:"memory"`
	Scope     MemoryScope `json:"scope"`
	FilePath  string      `json:"filePath,omitempty"`
	Mandatory bool        `json:"mandatory"`
}

func rankCandidates(request CapsuleRequest) []CapsuleCandidate {
	deduplicated := make(map[string]scoredCandidate)
	for _, candidate := range request.Candidates {
		if !eligibleForCapsule(candidate, request) {
			continue
		}
		key := normalizedMemoryText(candidate.Memory.Text)
		if key == "" {
			continue
		}
		scored := scoredCandidate{CapsuleCandidate: candidate, score: candidateScore(candidate, request)}
		if existing, ok := deduplicated[key]; !ok || scored.score > existing.score ||
			(scored.score == existing.score && scored.Memory.ID < existing.Memory.ID) {
			deduplicated[key] = scored
		}
	}
	result := make([]scoredCandidate, 0, len(deduplicated))
	for _, candidate := range deduplicated {
		result = append(result, candidate)
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].score != result[right].score {
			return result[left].score > result[right].score
		}
		return result[left].Memory.ID < result[right].Memory.ID
	})
	ranked := make([]CapsuleCandidate, len(result))
	for index, candidate := range result {
		ranked[index] = candidate.CapsuleCandidate
	}
	return ranked
}

type scoredCandidate struct {
	CapsuleCandidate
	score float64
}

func eligibleForCapsule(candidate CapsuleCandidate, request CapsuleRequest) bool {
	memory := candidate.Memory
	if memory.State != MemoryActive {
		return false
	}
	if memory.ExpiresAt != nil && !memory.ExpiresAt.After(request.Now) {
		return false
	}
	if memory.ProjectID != "" && memory.ProjectID != request.ProjectID {
		return false
	}
	return true
}

func candidateScore(candidate CapsuleCandidate, request CapsuleRequest) float64 {
	score := candidate.Memory.Confidence * 100
	if candidate.Mandatory {
		score += 1000
	}
	score += float64(sourcePriority(candidate.Memory.Source) * 100)
	switch candidate.Scope {
	case ScopeFile:
		if request.CurrentFile != "" && sameFile(candidate.FilePath, request.CurrentFile) {
			score += 350
		} else {
			score += 150
		}
	case ScopeProject:
		score += 250
	case ScopeGlobal:
		score += 25
	}
	return score
}

func sameFile(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func normalizedMemoryText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}
