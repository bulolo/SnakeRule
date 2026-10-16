package repo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

type RulesetRepository interface {
	List(context.Context) ([]domain.Ruleset, error)
	Get(context.Context, string) (domain.Ruleset, error)
	GetPublished(context.Context, string) (domain.Ruleset, error)
	ListVersions(context.Context, string) ([]domain.RulesetVersion, error)
	GetVersion(context.Context, string, int) (domain.RulesetVersion, error)
	Publish(context.Context, string, string, string) (domain.RulesetVersion, error)
	Rollback(context.Context, string, int, string) (domain.RulesetVersion, error)
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

func (r *PersistentRulesetRepository) GetPublished(ctx context.Context, id string) (domain.Ruleset, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return domain.Ruleset{}, err
	}
	if item.PublishedVersion == 0 {
		return domain.Ruleset{}, ErrNotFound
	}
	version, err := r.GetVersion(ctx, id, item.PublishedVersion)
	if err != nil {
		return domain.Ruleset{}, err
	}
	return version.Snapshot, nil
}

func (r *PersistentRulesetRepository) ListVersions(ctx context.Context, id string) ([]domain.RulesetVersion, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	db := currentDatabase()
	if db == nil {
		return nil, fmt.Errorf("PostgreSQL is not initialized")
	}
	rows, err := db.QueryContext(ctx, `SELECT version_number,snapshot,content_hash,created_by,created_at,change_note FROM ruleset_versions WHERE organization_id=$1 AND project_id=$2 AND ruleset_id=$3 ORDER BY version_number DESC`, organizationIDFrom(ctx), projectIDFrom(ctx), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := make([]domain.RulesetVersion, 0)
	for rows.Next() {
		version := domain.RulesetVersion{RulesetID: id}
		var snapshot []byte
		if err := rows.Scan(&version.Version, &snapshot, &version.ContentHash, &version.CreatedBy, &version.CreatedAt, &version.ChangeNote); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &version.Snapshot); err != nil {
			return nil, err
		}
		version.Snapshot.PublishedVersion = version.Version
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (r *PersistentRulesetRepository) GetVersion(ctx context.Context, id string, number int) (domain.RulesetVersion, error) {
	db := currentDatabase()
	if db == nil {
		return domain.RulesetVersion{}, fmt.Errorf("PostgreSQL is not initialized")
	}
	version := domain.RulesetVersion{RulesetID: id}
	var snapshot []byte
	err := db.QueryRowContext(ctx, `SELECT version_number,snapshot,content_hash,created_by,created_at,change_note FROM ruleset_versions WHERE organization_id=$1 AND project_id=$2 AND ruleset_id=$3 AND version_number=$4`, organizationIDFrom(ctx), projectIDFrom(ctx), id, number).Scan(&version.Version, &snapshot, &version.ContentHash, &version.CreatedBy, &version.CreatedAt, &version.ChangeNote)
	if err == sql.ErrNoRows {
		return domain.RulesetVersion{}, ErrNotFound
	}
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	if err := json.Unmarshal(snapshot, &version.Snapshot); err != nil {
		return domain.RulesetVersion{}, err
	}
	version.Snapshot.PublishedVersion = version.Version
	return version, nil
}

func (r *PersistentRulesetRepository) Publish(ctx context.Context, id, actor, note string) (domain.RulesetVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	scope := rulesetScopeID(ctx)
	item, ok := r.items[scope][id]
	if !ok {
		return domain.RulesetVersion{}, ErrNotFound
	}
	return r.publishLocked(ctx, item, actor, note)
}

func (r *PersistentRulesetRepository) Rollback(ctx context.Context, id string, number int, actor string) (domain.RulesetVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	scope := rulesetScopeID(ctx)
	if _, ok := r.items[scope][id]; !ok {
		return domain.RulesetVersion{}, ErrNotFound
	}
	historical, err := r.GetVersion(ctx, id, number)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	return r.publishLocked(ctx, historical.Snapshot, actor, fmt.Sprintf("Rollback to version %d", number))
}

func (r *PersistentRulesetRepository) publishLocked(ctx context.Context, item domain.Ruleset, actor, note string) (domain.RulesetVersion, error) {
	scope := rulesetScopeID(ctx)
	id := item.ID
	db := currentDatabase()
	if db == nil {
		return domain.RulesetVersion{}, fmt.Errorf("PostgreSQL is not initialized")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	defer func() { _ = tx.Rollback() }()
	lockKey := scope + "/" + id
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return domain.RulesetVersion{}, err
	}
	var number int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_number),0)+1 FROM ruleset_versions WHERE organization_id=$1 AND project_id=$2 AND ruleset_id=$3`, organizationIDFrom(ctx), projectIDFrom(ctx), id).Scan(&number); err != nil {
		return domain.RulesetVersion{}, err
	}
	item.PublishedVersion = number
	snapshot, err := json.Marshal(item)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	content := item
	content.PublishedVersion = 0
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(contentBytes))
	createdAt := time.Now().UTC()
	if actor == "" {
		actor = "unknown"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ruleset_versions(organization_id,project_id,ruleset_id,version_number,snapshot,content_hash,created_by,created_at,change_note) VALUES($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9)`, organizationIDFrom(ctx), projectIDFrom(ctx), id, number, snapshot, hash, actor, createdAt, strings.TrimSpace(note)); err != nil {
		return domain.RulesetVersion{}, err
	}
	itemMap := r.items[scope]
	previous := itemMap[id]
	itemMap[id] = item
	payload, err := json.Marshal(r.items)
	if err == nil {
		err = syncEntity(tx, r.key, string(payload))
	}
	if err != nil {
		itemMap[id] = previous
		return domain.RulesetVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		itemMap[id] = previous
		return domain.RulesetVersion{}, err
	}
	return domain.RulesetVersion{RulesetID: id, Version: number, Snapshot: item, ContentHash: hash, CreatedBy: actor, CreatedAt: createdAt, ChangeNote: strings.TrimSpace(note)}, nil
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
	if err := saveEntity(r.key, r.items); err != nil {
		return err
	}
	_, err := currentDatabase().ExecContext(ctx, `DELETE FROM ruleset_versions WHERE organization_id=$1 AND project_id=$2 AND ruleset_id=$3`, organizationIDFrom(ctx), projectIDFrom(ctx), id)
	return err
}

func (r *PersistentRulesetRepository) DeleteProjectScope(organizationID, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, organizationID+"/"+projectID)
	if err := saveEntity(r.key, r.items); err != nil {
		return err
	}
	_, err := currentDatabase().Exec(`DELETE FROM ruleset_versions WHERE organization_id=$1 AND project_id=$2`, organizationID, projectID)
	return err
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
