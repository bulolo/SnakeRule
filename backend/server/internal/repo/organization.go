package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

const DefaultOrganizationID = "default"

type organizationContextKey struct{}
type projectContextKey struct{}

func WithOrganization(ctx context.Context, id string) context.Context {
	if id == "" {
		id = DefaultOrganizationID
	}
	return context.WithValue(ctx, organizationContextKey{}, id)
}

func organizationIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(organizationContextKey{}).(string)
	if id == "" {
		return DefaultOrganizationID
	}
	return id
}

func OrganizationIDFrom(ctx context.Context) string { return organizationIDFrom(ctx) }

func WithProject(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, projectContextKey{}, id)
}

func projectIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(projectContextKey{}).(string)
	return id
}

func ProjectIDFrom(ctx context.Context) string { return projectIDFrom(ctx) }

type OrganizationRepository interface {
	List(context.Context) ([]domain.Organization, error)
	Get(context.Context, string) (domain.Organization, error)
	Save(context.Context, domain.Organization) error
	Delete(context.Context, string) error
}

type PersistentOrganizationRepository struct {
	mu    sync.RWMutex
	items map[string]domain.Organization
	path  string
}

func newOrganizationRepository() *PersistentOrganizationRepository {
	now := time.Now().UTC()
	return &PersistentOrganizationRepository{items: map[string]domain.Organization{
		DefaultOrganizationID: {ID: DefaultOrganizationID, Name: "默认组织", Slug: "default", Description: "SnakeRule 初始组织", CreatedAt: now, UpdatedAt: now},
	}}
}

func NewPersistentOrganizationRepository(path string) (*PersistentOrganizationRepository, error) {
	r := newOrganizationRepository()
	r.path = path
	if err := loadPersistent(path, &r.items); err != nil {
		return nil, err
	}
	if r.items == nil {
		r = newOrganizationRepository()
		r.path = path
	}
	if _, ok := r.items[DefaultOrganizationID]; !ok {
		now := time.Now().UTC()
		r.items[DefaultOrganizationID] = domain.Organization{ID: DefaultOrganizationID, Name: "默认组织", Slug: "default", Description: "SnakeRule 初始组织", CreatedAt: now, UpdatedAt: now}
	}
	// Persist seeded defaults too, so a brand-new database has a durable org.
	if err := savePersistent(path, r.items); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *PersistentOrganizationRepository) List(context.Context) ([]domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Organization, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID == DefaultOrganizationID {
			return items[j].ID != DefaultOrganizationID
		}
		if items[j].ID == DefaultOrganizationID {
			return false
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (r *PersistentOrganizationRepository) Get(_ context.Context, id string) (domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.Organization{}, ErrNotFound
	}
	return item, nil
}

func (r *PersistentOrganizationRepository) Save(_ context.Context, item domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return savePersistent(r.path, r.items)
}

func (r *PersistentOrganizationRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == DefaultOrganizationID {
		return ErrProtected
	}
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return savePersistent(r.path, r.items)
}
