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
}

// MemoryAlertChannelRepository is suitable for the single-process starter only.
type MemoryAlertChannelRepository struct {
	mu    sync.RWMutex
	items map[string]domain.AlertChannel
	path  string
}

type persistedAlertChannel struct {
	Channel domain.AlertChannel `json:"channel"`
	Secret  string              `json:"secret,omitempty"`
}

func NewMemoryAlertChannelRepository() *MemoryAlertChannelRepository {
	return &MemoryAlertChannelRepository{items: make(map[string]domain.AlertChannel)}
}

func NewPersistentAlertChannelRepository(path string) (*MemoryAlertChannelRepository, error) {
	r := NewMemoryAlertChannelRepository()
	r.path = path
	var stored map[string]persistedAlertChannel
	if err := loadJSONFile(path, &stored); err != nil {
		return nil, err
	}
	for id, item := range stored {
		item.Channel.Secret = item.Secret
		r.items[id] = item.Channel
	}
	return r, nil
}

func (r *MemoryAlertChannelRepository) persist() error {
	stored := make(map[string]persistedAlertChannel, len(r.items))
	for id, item := range r.items {
		secret := item.Secret
		item.Secret = ""
		item.HasSecret = secret != ""
		stored[id] = persistedAlertChannel{Channel: item, Secret: secret}
	}
	return saveJSONFile(r.path, stored)
}

func (r *MemoryAlertChannelRepository) List(context.Context) ([]domain.AlertChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.AlertChannel, 0, len(r.items))
	for _, item := range r.items {
		item.HasSecret = item.Secret != ""
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryAlertChannelRepository) Get(_ context.Context, id string) (domain.AlertChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.AlertChannel{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryAlertChannelRepository) Save(_ context.Context, item domain.AlertChannel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return r.persist()
}

func (r *MemoryAlertChannelRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return r.persist()
}
