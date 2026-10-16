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

// PersistentRulesetRepository keeps an in-process cache backed by the configured persistent store.
type PersistentRulesetRepository struct {
	mu    sync.RWMutex
	items map[string]map[string]domain.Ruleset
	key   string
}

func newRulesetRepository() *PersistentRulesetRepository {
	return &PersistentRulesetRepository{items: make(map[string]map[string]domain.Ruleset)}
}

func NewPersistentRulesetRepository(key string) (*PersistentRulesetRepository, error) {
	r := newRulesetRepository()
	r.key = key
	if err := loadEntity(key, &r.items); err != nil {
		return nil, err
	}
	if r.items == nil {
		r.items = make(map[string]map[string]domain.Ruleset)
	}
	return r, nil
}

func (r *PersistentRulesetRepository) List(ctx context.Context) ([]domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	organizationItems := r.items[rulesetScopeID(ctx)]
	items := make([]domain.Ruleset, 0, len(organizationItems))
	for _, item := range organizationItems {
		items = append(items, cloneRuleset(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}
func (r *PersistentRulesetRepository) Get(ctx context.Context, id string) (domain.Ruleset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[rulesetScopeID(ctx)][id]
	if !ok {
		return domain.Ruleset{}, ErrNotFound
	}
	return cloneRuleset(item), nil
}
func (r *PersistentRulesetRepository) Save(ctx context.Context, item domain.Ruleset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	scopeID := rulesetScopeID(ctx)
	if r.items[scopeID] == nil {
		r.items[scopeID] = make(map[string]domain.Ruleset)
	}
	r.items[scopeID][item.ID] = cloneRuleset(item)
	return saveEntity(r.key, r.items)
}
func (r *PersistentRulesetRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.items[rulesetScopeID(ctx)]
	if _, ok := items[id]; !ok {
		return ErrNotFound
	}
	delete(items, id)
	return saveEntity(r.key, r.items)
}

func (r *PersistentRulesetRepository) DeleteProjectScope(organizationID, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, organizationID+"/"+projectID)
	return saveEntity(r.key, r.items)
}

func rulesetScopeID(ctx context.Context) string {
	return organizationIDFrom(ctx) + "/" + projectIDFrom(ctx)
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
