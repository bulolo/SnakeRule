package repo

import (
	"context"
	"sort"
	"sync"

	"github.com/szplanner/snakerule/internal/domain"
)

type AlertChannelRepository interface {
	List(context.Context) ([]domain.AlertChannel, error)
	Get(context.Context, string) (domain.AlertChannel, error)
	Save(context.Context, domain.AlertChannel) error
	Delete(context.Context, string) error
	DeleteOrganization(context.Context, string) error
}

// PersistentAlertChannelRepository keeps an in-process cache backed by the configured persistent store.
type PersistentAlertChannelRepository struct {
	mu    sync.RWMutex
	items map[string]domain.AlertChannel
	path  string
}

type persistedAlertChannel struct {
	Channel domain.AlertChannel `json:"channel"`
	Secret  string              `json:"secret,omitempty"`
}

func newAlertChannelRepository() *PersistentAlertChannelRepository {
	return &PersistentAlertChannelRepository{items: make(map[string]domain.AlertChannel)}
}

func NewPersistentAlertChannelRepository(path string) (*PersistentAlertChannelRepository, error) {
	r := newAlertChannelRepository()
	r.path = path
	var stored map[string]persistedAlertChannel
	if err := loadPersistent(path, &stored); err != nil {
		return nil, err
	}
	for id, item := range stored {
		if item.Channel.OrganizationID == "" && id != "internal-inbox" {
			item.Channel.OrganizationID = DefaultOrganizationID
		}
		item.Channel.Secret = item.Secret
		r.items[id] = item.Channel
	}
	return r, nil
}

func (r *PersistentAlertChannelRepository) persist() error {
	stored := make(map[string]persistedAlertChannel, len(r.items))
	for id, item := range r.items {
		secret := item.Secret
		item.Secret = ""
		item.HasSecret = secret != ""
		stored[id] = persistedAlertChannel{Channel: item, Secret: secret}
	}
	return savePersistent(r.path, stored)
}

func (r *PersistentAlertChannelRepository) List(ctx context.Context) ([]domain.AlertChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.AlertChannel, 0, len(r.items))
	for _, item := range r.items {
		if item.OrganizationID != "" && item.OrganizationID != OrganizationIDFrom(ctx) {
			continue
		}
		item.HasSecret = item.Secret != ""
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (r *PersistentAlertChannelRepository) Get(ctx context.Context, id string) (domain.AlertChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok || (item.OrganizationID != "" && item.OrganizationID != OrganizationIDFrom(ctx)) {
		return domain.AlertChannel{}, ErrNotFound
	}
	return item, nil
}

func (r *PersistentAlertChannelRepository) Save(_ context.Context, item domain.AlertChannel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return r.persist()
}

func (r *PersistentAlertChannelRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	if item.OrganizationID != "" && item.OrganizationID != OrganizationIDFrom(ctx) {
		return ErrNotFound
	}
	delete(r.items, id)
	return r.persist()
}

func (r *PersistentAlertChannelRepository) DeleteOrganization(_ context.Context, organizationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, item := range r.items {
		if item.OrganizationID == organizationID {
			delete(r.items, id)
		}
	}
	return r.persist()
}
