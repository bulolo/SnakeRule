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
	items map[string]map[string]domain.Ruleset
}

func NewMemoryRulesetRepository() *MemoryRulesetRepository {
	return &MemoryRulesetRepository{items: make(map[string]map[string]domain.Ruleset)}
}

func (r *MemoryRulesetRepository) List(ctx context.Context) ([]domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	workspaceItems := r.items[workspaceIDFrom(ctx)]
	items := make([]domain.Ruleset, 0, len(workspaceItems))
	for _, item := range workspaceItems {
		items = append(items, cloneRuleset(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}
func (r *MemoryRulesetRepository) Get(ctx context.Context, id string) (domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[workspaceIDFrom(ctx)][id]
	if !ok {
		return domain.Ruleset{}, ErrNotFound
	}
	return cloneRuleset(item), nil
}
func (r *MemoryRulesetRepository) Save(ctx context.Context, item domain.Ruleset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	workspaceID := workspaceIDFrom(ctx)
	if r.items[workspaceID] == nil {
		r.items[workspaceID] = make(map[string]domain.Ruleset)
	}
	r.items[workspaceID][item.ID] = cloneRuleset(item)
	return nil
}
func (r *MemoryRulesetRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.items[workspaceIDFrom(ctx)]
	if _, ok := items[id]; !ok {
		return ErrNotFound
	}
	delete(items, id)
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
