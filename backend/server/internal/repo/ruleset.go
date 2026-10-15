package repo

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/szplanner/snakerule/internal/domain"
)

type RulesetRepository interface {
	List(context.Context) ([]domain.Ruleset, error)
	Get(context.Context, string) (domain.Ruleset, error)
	Save(context.Context, domain.Ruleset) error
	Delete(context.Context, string) error
}

// MemoryRulesetRepository is a development adapter. It is intentionally not durable.
type MemoryRulesetRepository struct {
	mu    sync.RWMutex
	items map[string]domain.Ruleset
}

func NewMemoryRulesetRepository() *MemoryRulesetRepository {
	return &MemoryRulesetRepository{items: make(map[string]domain.Ruleset)}
}

func (r *MemoryRulesetRepository) List(_ context.Context) ([]domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Ruleset, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, cloneRuleset(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}
func (r *MemoryRulesetRepository) Get(_ context.Context, id string) (domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.Ruleset{}, ErrNotFound
	}
	return cloneRuleset(item), nil
}
func (r *MemoryRulesetRepository) Save(_ context.Context, item domain.Ruleset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = cloneRuleset(item)
	return nil
}
func (r *MemoryRulesetRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func cloneRuleset(item domain.Ruleset) domain.Ruleset {
	data, err := json.Marshal(item)
	if err != nil {
		return item
	}
	var copy domain.Ruleset
	if err := json.Unmarshal(data, &copy); err != nil {
		return item
	}
	return copy
}
