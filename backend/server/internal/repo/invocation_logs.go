package repo

import (
	"sort"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

type MemoryInvocationLogRepository struct {
	mu    sync.RWMutex
	items []domain.InvocationLog
	path  string
}

func NewPersistentInvocationLogRepository(path string) (*MemoryInvocationLogRepository, error) {
	r := &MemoryInvocationLogRepository{path: path}
	if err := loadJSONFile(path, &r.items); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *MemoryInvocationLogRepository) Add(item domain.InvocationLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item.ID == "" {
		item.ID = time.Now().UTC().Format("20060102150405.000000000")
	}
	r.items = append(r.items, item)
	if len(r.items) > 2000 {
		r.items = append([]domain.InvocationLog(nil), r.items[len(r.items)-2000:]...)
	}
	return saveJSONFile(r.path, r.items)
}

func (r *MemoryInvocationLogRepository) List(workspaceID, projectID string) []domain.InvocationLog {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.InvocationLog, 0)
	for _, item := range r.items {
		if item.WorkspaceID == workspaceID && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (r *MemoryInvocationLogRepository) DeleteProject(workspaceID, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.items[:0]
	for _, item := range r.items {
		if item.WorkspaceID != workspaceID || item.ProjectID != projectID {
			kept = append(kept, item)
		}
	}
	r.items = kept
	return saveJSONFile(r.path, r.items)
}
