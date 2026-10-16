package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

const DefaultWorkspaceID = "default"

type workspaceContextKey struct{}
type projectContextKey struct{}

func WithWorkspace(ctx context.Context, id string) context.Context {
	if id == "" {
		id = DefaultWorkspaceID
	}
	return context.WithValue(ctx, workspaceContextKey{}, id)
}

func workspaceIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(workspaceContextKey{}).(string)
	if id == "" {
		return DefaultWorkspaceID
	}
	return id
}

func WorkspaceIDFrom(ctx context.Context) string { return workspaceIDFrom(ctx) }

func WithProject(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, projectContextKey{}, id)
}

func projectIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(projectContextKey{}).(string)
	return id
}

func ProjectIDFrom(ctx context.Context) string { return projectIDFrom(ctx) }

type WorkspaceRepository interface {
	List(context.Context) ([]domain.Workspace, error)
	Get(context.Context, string) (domain.Workspace, error)
	Save(context.Context, domain.Workspace) error
	Delete(context.Context, string) error
}

type MemoryWorkspaceRepository struct {
	mu    sync.RWMutex
	items map[string]domain.Workspace
	path  string
}

func NewMemoryWorkspaceRepository() *MemoryWorkspaceRepository {
	now := time.Now().UTC()
	return &MemoryWorkspaceRepository{items: map[string]domain.Workspace{
		DefaultWorkspaceID: {ID: DefaultWorkspaceID, Name: "默认工作空间", Slug: "default", Description: "SnakeRule 初始工作空间", CreatedAt: now, UpdatedAt: now},
	}}
}

func NewPersistentWorkspaceRepository(path string) (*MemoryWorkspaceRepository, error) {
	r := NewMemoryWorkspaceRepository()
	r.path = path
	if err := loadJSONFile(path, &r.items); err != nil {
		return nil, err
	}
	if r.items == nil {
		r = NewMemoryWorkspaceRepository()
		r.path = path
	}
	if _, ok := r.items[DefaultWorkspaceID]; !ok {
		now := time.Now().UTC()
		r.items[DefaultWorkspaceID] = domain.Workspace{ID: DefaultWorkspaceID, Name: "默认工作空间", Slug: "default", Description: "SnakeRule 初始工作空间", CreatedAt: now, UpdatedAt: now}
		if err := saveJSONFile(path, r.items); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *MemoryWorkspaceRepository) List(context.Context) ([]domain.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Workspace, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID == DefaultWorkspaceID {
			return items[j].ID != DefaultWorkspaceID
		}
		if items[j].ID == DefaultWorkspaceID {
			return false
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (r *MemoryWorkspaceRepository) Get(_ context.Context, id string) (domain.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.Workspace{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryWorkspaceRepository) Save(_ context.Context, item domain.Workspace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return saveJSONFile(r.path, r.items)
}

func (r *MemoryWorkspaceRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == DefaultWorkspaceID {
		return ErrProtected
	}
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return saveJSONFile(r.path, r.items)
}
