package repo

import (
	"sort"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

type PersistentNotificationRepository struct {
	mu    sync.Mutex
	items []domain.Notification
	key   string
}

func NewPersistentNotificationRepository(key string) (*PersistentNotificationRepository, error) {
	r := &PersistentNotificationRepository{key: key}
	if err := loadEntity(key, &r.items); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *PersistentNotificationRepository) List(organizationID string) []domain.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Notification, 0, len(r.items))
	for _, item := range r.items {
		if item.OrganizationID == organizationID {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
func (r *PersistentNotificationRepository) Add(item domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item.ID == "" {
		item.ID = time.Now().UTC().Format("20060102150405.000000000")
	}
	r.items = append(r.items, item)
	if len(r.items) > 500 {
		r.items = append([]domain.Notification(nil), r.items[len(r.items)-500:]...)
	}
	return saveEntity(r.key, r.items)
}
func (r *PersistentNotificationRepository) MarkRead(organizationID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id && r.items[i].OrganizationID == organizationID {
			r.items[i].Read = true
			return saveEntity(r.key, r.items)
		}
	}
	return ErrNotFound
}
func (r *PersistentNotificationRepository) MarkAllRead(organizationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].OrganizationID == organizationID {
			r.items[i].Read = true
		}
	}
	return saveEntity(r.key, r.items)
}
