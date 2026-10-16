package repo

import (
	"context"
	"sort"
	"sync"

	"github.com/szplanner/snakerule/internal/domain"
)

type ProjectKeyRepository interface {
	List(context.Context, string) ([]domain.APIKeyCredential, error)
	Save(context.Context, domain.APIKeyCredential) error
	Delete(context.Context, string, string) error
	FindByHash(context.Context, string) (domain.APIKeyCredential, error)
}

type MemoryProjectKeyRepository struct {
	mu    sync.RWMutex
	items map[string]domain.APIKeyCredential
	path  string
}

type persistedProjectKey struct {
	Credential               domain.APIKeyCredential `json:"credential"`
	Hash, Workspace, Project string
}

func NewMemoryProjectKeyRepository() *MemoryProjectKeyRepository {
	return &MemoryProjectKeyRepository{items: map[string]domain.APIKeyCredential{}}
}
func NewPersistentProjectKeyRepository(path string) (*MemoryProjectKeyRepository, error) {
	r := NewMemoryProjectKeyRepository()
	r.path = path
	var stored map[string]persistedProjectKey
	if err := loadJSONFile(path, &stored); err != nil {
		return nil, err
	}
	for id, item := range stored {
		c := item.Credential
		c.Hash = item.Hash
		c.Workspace = item.Workspace
		c.Project = item.Project
		r.items[id] = c
	}
	return r, nil
}
func (r *MemoryProjectKeyRepository) persist() error {
	stored := make(map[string]persistedProjectKey, len(r.items))
	for id, c := range r.items {
		hash, workspace, project := c.Hash, c.Workspace, c.Project
		c.Hash = ""
		c.Workspace = ""
		c.Project = ""
		stored[id] = persistedProjectKey{Credential: c, Hash: hash, Workspace: workspace, Project: project}
	}
	return saveJSONFile(r.path, stored)
}
func (r *MemoryProjectKeyRepository) List(ctx context.Context, project string) ([]domain.APIKeyCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.APIKeyCredential, 0)
	for _, item := range r.items {
		if item.Workspace == workspaceIDFrom(ctx) && item.Project == project {
			item.Hash = ""
			if item.Type == "" {
				item.Type = "apikey"
			}
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
func (r *MemoryProjectKeyRepository) Save(_ context.Context, item domain.APIKeyCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return r.persist()
}
func (r *MemoryProjectKeyRepository) Delete(ctx context.Context, project, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Workspace != workspaceIDFrom(ctx) || item.Project != project {
		return ErrNotFound
	}
	delete(r.items, id)
	return r.persist()
}
func (r *MemoryProjectKeyRepository) FindByHash(_ context.Context, hash string) (domain.APIKeyCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if item.Hash == hash {
			return item, nil
		}
	}
	return domain.APIKeyCredential{}, ErrNotFound
}
