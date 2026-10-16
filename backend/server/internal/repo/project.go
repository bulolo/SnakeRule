package repo

import (
	"context"
	"sort"
	"sync"

	"github.com/szplanner/snakerule/internal/domain"
)

type ProjectRepository interface {
	List(context.Context) ([]domain.Project, error)
	Get(context.Context, string) (domain.Project, error)
	Save(context.Context, domain.Project) error
	Delete(context.Context, string) error
}

type MemoryProjectRepository struct {
	mu    sync.RWMutex
	items map[string]map[string]domain.Project
	path  string
}

func NewMemoryProjectRepository() *MemoryProjectRepository {
	return &MemoryProjectRepository{items: map[string]map[string]domain.Project{
		DefaultWorkspaceID: {DefaultProjectID: {ID: DefaultProjectID, Name: "默认项目", Slug: "default"}},
	}}
}

func NewPersistentProjectRepository(path string) (*MemoryProjectRepository, error) {
	r := NewMemoryProjectRepository()
	r.path = path
	if err := loadJSONFile(path, &r.items); err != nil {
		return nil, err
	}
	if r.items == nil {
		r = NewMemoryProjectRepository()
		r.path = path
	}
	return r, nil
}

func (r *MemoryProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.items[workspaceIDFrom(ctx)]) == 0 {
		return []domain.Project{cloneProject(domain.Project{ID: DefaultProjectID, Name: "默认项目", Slug: "default"})}, nil
	}
	items := make([]domain.Project, 0, len(r.items[workspaceIDFrom(ctx)]))
	for _, item := range r.items[workspaceIDFrom(ctx)] {
		items = append(items, cloneProject(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID == DefaultProjectID {
			return true
		}
		if items[j].ID == DefaultProjectID {
			return false
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (r *MemoryProjectRepository) Get(ctx context.Context, id string) (domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[workspaceIDFrom(ctx)][id]
	if !ok && id == DefaultProjectID {
		return cloneProject(domain.Project{ID: DefaultProjectID, Name: "默认项目", Slug: "default"}), nil
	}
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return cloneProject(item), nil
}

func (r *MemoryProjectRepository) Save(ctx context.Context, item domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	wid := workspaceIDFrom(ctx)
	if r.items[wid] == nil {
		r.items[wid] = map[string]domain.Project{DefaultProjectID: {ID: DefaultProjectID, Name: "默认项目", Slug: "default"}}
	}
	r.items[wid][item.ID] = cloneProject(item)
	return saveJSONFile(r.path, r.items)
}

func (r *MemoryProjectRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == DefaultProjectID {
		return ErrProtected
	}
	projects := r.items[workspaceIDFrom(ctx)]
	if _, ok := projects[id]; !ok {
		return ErrNotFound
	}
	delete(projects, id)
	return saveJSONFile(r.path, r.items)
}

func cloneProject(item domain.Project) domain.Project {
	return item
}
