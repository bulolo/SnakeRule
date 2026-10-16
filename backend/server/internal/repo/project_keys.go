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
	FindByID(context.Context, string) (domain.APIKeyCredential, error)
}

type PersistentProjectKeyRepository struct {
	mu    sync.RWMutex
	items map[string]domain.APIKeyCredential
	path  string
}

type persistedProjectKey struct {
	Credential                                          domain.APIKeyCredential `json:"credential"`
	Hash, Organization, Project, SigningSecretEncrypted string
}

func newProjectKeyRepository() *PersistentProjectKeyRepository {
	return &PersistentProjectKeyRepository{items: map[string]domain.APIKeyCredential{}}
}
func NewPersistentProjectKeyRepository(path string) (*PersistentProjectKeyRepository, error) {
	r := newProjectKeyRepository()
	r.path = path
	var stored map[string]persistedProjectKey
	if err := loadPersistent(path, &stored); err != nil {
		return nil, err
	}
	for id, item := range stored {
		c := item.Credential
		c.Hash = item.Hash
		c.Organization = item.Organization
		c.Project = item.Project
		c.SigningSecretEncrypted = item.SigningSecretEncrypted
		r.items[id] = c
	}
	return r, nil
}
func (r *PersistentProjectKeyRepository) persist() error {
	stored := make(map[string]persistedProjectKey, len(r.items))
	for id, c := range r.items {
		hash, organization, project := c.Hash, c.Organization, c.Project
		c.Hash = ""
		c.Organization = ""
		c.Project = ""
		encrypted := c.SigningSecretEncrypted
		c.SigningSecretEncrypted = ""
		stored[id] = persistedProjectKey{Credential: c, Hash: hash, Organization: organization, Project: project, SigningSecretEncrypted: encrypted}
	}
	return savePersistent(r.path, stored)
}
func (r *PersistentProjectKeyRepository) List(ctx context.Context, project string) ([]domain.APIKeyCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.APIKeyCredential, 0)
	for _, item := range r.items {
		if item.Organization == organizationIDFrom(ctx) && item.Project == project {
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
func (r *PersistentProjectKeyRepository) Save(_ context.Context, item domain.APIKeyCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return r.persist()
}
func (r *PersistentProjectKeyRepository) Delete(ctx context.Context, project, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Organization != organizationIDFrom(ctx) || item.Project != project {
		return ErrNotFound
	}
	delete(r.items, id)
	return r.persist()
}

func (r *PersistentProjectKeyRepository) DeleteProject(organizationID, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := false
	for id, item := range r.items {
		if item.Organization == organizationID && item.Project == projectID {
			delete(r.items, id)
			changed = true
		}
	}
	if changed {
		return r.persist()
	}
	return nil
}
func (r *PersistentProjectKeyRepository) FindByHash(_ context.Context, hash string) (domain.APIKeyCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if item.Hash == hash {
			return item, nil
		}
	}
	return domain.APIKeyCredential{}, ErrNotFound
}

func (r *PersistentProjectKeyRepository) FindByID(_ context.Context, id string) (domain.APIKeyCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.APIKeyCredential{}, ErrNotFound
	}
	return item, nil
}
