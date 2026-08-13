package context

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryState string

const (
	MemoryCandidate MemoryState = "candidate"
	MemoryActive    MemoryState = "active"
	MemoryDisabled  MemoryState = "disabled"
	MemoryDeleted   MemoryState = "deleted"
)

type MemorySource string

const (
	SourceUserExplicit   MemorySource = "user-explicit"
	SourceRepositoryRule MemorySource = "repository-rule"
	SourceInferred       MemorySource = "inferred"
)

var (
	ErrMemoryDeleted  = errors.New("memory has been explicitly deleted")
	ErrMemoryDisabled = errors.New("memory has been disabled")
	ErrMemoryNotFound = errors.New("memory not found")
)

type Memory struct {
	ID               string       `json:"id"`
	ProjectID        string       `json:"projectId"`
	Text             string       `json:"text"`
	Source           MemorySource `json:"source"`
	Confidence       float64      `json:"confidence"`
	ObservationCount int          `json:"observationCount"`
	State            MemoryState  `json:"state"`
	ExpiresAt        *time.Time   `json:"expiresAt,omitempty"`
}

type Observation struct {
	ID        string
	ProjectID string
	Text      string
	Source    MemorySource
	ExpiresAt *time.Time
}

type MemoryManager struct {
	mu        sync.RWMutex
	threshold int
	memories  map[string]Memory
	deleted   map[string]struct{}
	cache     map[string][]Memory
}

func NewMemoryManager(inferenceThreshold int) *MemoryManager {
	if inferenceThreshold < 2 {
		inferenceThreshold = 3
	}
	return &MemoryManager{
		threshold: inferenceThreshold,
		memories:  make(map[string]Memory),
		deleted:   make(map[string]struct{}),
		cache:     make(map[string][]Memory),
	}
}

func (manager *MemoryManager) Observe(observation Observation) (Memory, error) {
	if err := validateObservation(observation); err != nil {
		return Memory{}, err
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if _, deleted := manager.deleted[observation.ID]; deleted {
		return Memory{}, ErrMemoryDeleted
	}

	memory, exists := manager.memories[observation.ID]
	if exists && memory.State == MemoryDisabled {
		return cloneMemory(memory), ErrMemoryDisabled
	}
	if exists && memory.ProjectID != observation.ProjectID {
		return Memory{}, fmt.Errorf("memory id belongs to another project")
	}
	if !exists {
		memory = Memory{ID: observation.ID, ProjectID: observation.ProjectID}
	}
	if !exists || sourcePriority(observation.Source) >= sourcePriority(memory.Source) {
		memory.Text = strings.TrimSpace(observation.Text)
		memory.Source = observation.Source
		memory.ExpiresAt = cloneTime(observation.ExpiresAt)
	}

	switch memory.Source {
	case SourceUserExplicit:
		memory.State = MemoryActive
		memory.Confidence = 1
		if memory.ObservationCount == 0 {
			memory.ObservationCount = 1
		}
	case SourceRepositoryRule:
		memory.State = MemoryActive
		memory.Confidence = 0.98
		if memory.ObservationCount == 0 {
			memory.ObservationCount = 1
		}
	case SourceInferred:
		memory.ObservationCount++
		memory.Confidence = minFloat(0.9, float64(memory.ObservationCount)/float64(manager.threshold)*0.75)
		if memory.ObservationCount >= manager.threshold {
			memory.State = MemoryActive
		} else {
			memory.State = MemoryCandidate
		}
	}
	manager.memories[memory.ID] = memory
	delete(manager.cache, memory.ProjectID)
	return cloneMemory(memory), nil
}

func (manager *MemoryManager) ListEligible(projectID string, now time.Time) []Memory {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	cached, ok := manager.cache[projectID]
	if !ok {
		cached = make([]Memory, 0)
		for _, memory := range manager.memories {
			if memory.ProjectID == projectID && memory.State == MemoryActive {
				cached = append(cached, cloneMemory(memory))
			}
		}
		sort.SliceStable(cached, func(left, right int) bool {
			leftPriority := sourcePriority(cached[left].Source)
			rightPriority := sourcePriority(cached[right].Source)
			if leftPriority != rightPriority {
				return leftPriority > rightPriority
			}
			if cached[left].Confidence != cached[right].Confidence {
				return cached[left].Confidence > cached[right].Confidence
			}
			return cached[left].ID < cached[right].ID
		})
		manager.cache[projectID] = cloneMemories(cached)
	}
	result := make([]Memory, 0, len(cached))
	for _, memory := range cached {
		if memory.ExpiresAt != nil && !memory.ExpiresAt.After(now) {
			continue
		}
		result = append(result, cloneMemory(memory))
	}
	return result
}

func (manager *MemoryManager) Disable(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	memory, ok := manager.memories[id]
	if !ok {
		return ErrMemoryNotFound
	}
	memory.State = MemoryDisabled
	manager.memories[id] = memory
	delete(manager.cache, memory.ProjectID)
	return nil
}

func (manager *MemoryManager) Enable(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	memory, ok := manager.memories[id]
	if !ok {
		return ErrMemoryNotFound
	}
	if memory.State == MemoryDeleted {
		return ErrMemoryDeleted
	}
	if memory.Source == SourceInferred && memory.ObservationCount < manager.threshold {
		memory.State = MemoryCandidate
	} else {
		memory.State = MemoryActive
	}
	manager.memories[id] = memory
	delete(manager.cache, memory.ProjectID)
	return nil
}

func (manager *MemoryManager) Delete(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	memory, ok := manager.memories[id]
	if !ok {
		return ErrMemoryNotFound
	}
	memory.State = MemoryDeleted
	memory.Text = ""
	manager.memories[id] = memory
	manager.deleted[id] = struct{}{}
	delete(manager.cache, memory.ProjectID)
	return nil
}

func validateObservation(observation Observation) error {
	if strings.TrimSpace(observation.ID) == "" || strings.TrimSpace(observation.ProjectID) == "" || strings.TrimSpace(observation.Text) == "" {
		return fmt.Errorf("memory id, project id, and text are required")
	}
	switch observation.Source {
	case SourceUserExplicit, SourceRepositoryRule, SourceInferred:
		return nil
	default:
		return fmt.Errorf("unsupported memory source %q", observation.Source)
	}
}

func sourcePriority(source MemorySource) int {
	switch source {
	case SourceUserExplicit:
		return 3
	case SourceRepositoryRule:
		return 2
	case SourceInferred:
		return 1
	default:
		return 0
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneMemory(memory Memory) Memory {
	memory.ExpiresAt = cloneTime(memory.ExpiresAt)
	return memory
}

func cloneMemories(memories []Memory) []Memory {
	result := make([]Memory, len(memories))
	for index, memory := range memories {
		result[index] = cloneMemory(memory)
	}
	return result
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
