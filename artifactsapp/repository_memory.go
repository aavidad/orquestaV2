package artifactsapp

import "sync"

type MemoryRepository struct {
	mu    sync.Mutex
	items []Artifact
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

func (r *MemoryRepository) SaveArtifact(artifact Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, artifact)
	return nil
}

func (r *MemoryRepository) ListArtifacts(filter ListFilter) ([]Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Artifact, 0, len(r.items))
	for _, item := range r.items {
		if matchesFilter(item, filter) {
			out = append(out, item)
		}
	}
	sortArtifacts(out)
	return out, nil
}

func matchesFilter(item Artifact, filter ListFilter) bool {
	if filter.Scope != "" && item.Scope != filter.Scope {
		return false
	}
	if filter.Kind != "" && item.Kind != filter.Kind {
		return false
	}
	if filter.TaskID != nil {
		if item.TaskID == nil || *item.TaskID != *filter.TaskID {
			return false
		}
	}
	if filter.Agent != "" && item.Agent != filter.Agent {
		return false
	}
	if filter.Project != "" && item.Project != filter.Project {
		return false
	}
	return true
}
