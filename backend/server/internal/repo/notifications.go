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
	path  string
}

func NewPersistentNotificationRepository(path string) (*PersistentNotificationRepository, error) {
	r := &PersistentNotificationRepository{path: path}
	if err := loadPersistent(path, &r.items); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *PersistentNotificationRepository) List() []domain.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Notification, len(r.items))
	copy(out, r.items)
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
	return savePersistent(r.path, r.items)
}
func (r *PersistentNotificationRepository) MarkRead(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].Read = true
			return savePersistent(r.path, r.items)
		}
	}
	return ErrNotFound
}
func (r *PersistentNotificationRepository) MarkAllRead() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		r.items[i].Read = true
	}
	return savePersistent(r.path, r.items)
}
