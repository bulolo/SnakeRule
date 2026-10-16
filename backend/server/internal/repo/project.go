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

type PersistentProjectRepository struct {
	mu    sync.RWMutex
	items map[string]map[string]domain.Project
	key   string
}

func newProjectRepository() *PersistentProjectRepository {
	return &PersistentProjectRepository{items: make(map[string]map[string]domain.Project)}
}

func NewPersistentProjectRepository(key string) (*PersistentProjectRepository, error) {
	r := newProjectRepository()
	r.key = key
	if err := loadEntity(key, &r.items); err != nil {
		return nil, err
	}
	if r.items == nil {
		r = newProjectRepository()
		r.key = key
	}
	return r, nil
}

func (r *PersistentProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Project, 0, len(r.items[organizationIDFrom(ctx)]))
	for _, item := range r.items[organizationIDFrom(ctx)] {
		items = append(items, cloneProject(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *PersistentProjectRepository) Get(ctx context.Context, id string) (domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[organizationIDFrom(ctx)][id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return cloneProject(item), nil
}

func (r *PersistentProjectRepository) Save(ctx context.Context, item domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	wid := organizationIDFrom(ctx)
	if r.items[wid] == nil {
		r.items[wid] = make(map[string]domain.Project)
	}
	r.items[wid][item.ID] = cloneProject(item)
	return saveEntity(r.key, r.items)
}

func (r *PersistentProjectRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	projects := r.items[organizationIDFrom(ctx)]
	if _, ok := projects[id]; !ok {
		return ErrNotFound
	}
	delete(projects, id)
	return saveEntity(r.key, r.items)
}

func cloneProject(item domain.Project) domain.Project {
	return item
}
