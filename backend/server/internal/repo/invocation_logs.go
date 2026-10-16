package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

type InvocationLogFilter struct {
	RulesetID string
	RequestID string
	Source    string
	Outcome   string
	Since     *time.Time
	Until     *time.Time
	Page      int
	Size      int
	Unpaged   bool
}

type InvocationLogPage struct {
	Items []domain.InvocationLog
	Total int
}

type PersistentInvocationLogRepository struct {
	cleanupMu sync.RWMutex
	cleanup   InvocationCleanupState
}

type InvocationCleanupState struct {
	LastRun      time.Time
	LastDeleted  int
	TotalDeleted int64
	LastError    string
}

func NewPersistentInvocationLogRepository(_ string) (*PersistentInvocationLogRepository, error) {
	if currentDatabase() == nil {
		return nil, fmt.Errorf("PostgreSQL must be initialized before creating invocation repository")
	}
	return &PersistentInvocationLogRepository{}, nil
}

func (r *PersistentInvocationLogRepository) Add(item domain.InvocationLog) error {
	db := currentDatabase()
	if db == nil {
		return fmt.Errorf("PostgreSQL is not initialized")
	}
	if item.ID == "" {
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return err
		}
		item.ID = hex.EncodeToString(id[:])
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	document, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO invocation_logs(id,organization_id,project_id,ruleset_id,ruleset_version,request_id,source,status_code,matched,created_at,document) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)`, item.ID, item.OrganizationID, item.ProjectID, item.RulesetID, item.RulesetVersion, item.RequestID, item.Source, item.StatusCode, item.Matched, item.CreatedAt, document)
	return err
}

func (r *PersistentInvocationLogRepository) List(organizationID, projectID string) []domain.InvocationLog {
	page, err := r.Query(context.Background(), organizationID, projectID, InvocationLogFilter{Unpaged: true})
	if err != nil {
		return []domain.InvocationLog{}
	}
	return page.Items
}

func (r *PersistentInvocationLogRepository) Query(ctx context.Context, organizationID, projectID string, filter InvocationLogFilter) (InvocationLogPage, error) {
	db := currentDatabase()
	if db == nil {
		return InvocationLogPage{}, fmt.Errorf("PostgreSQL is not initialized")
	}
	where := `organization_id=$1 AND project_id=$2 AND ($3='' OR ruleset_id=$3) AND ($4='' OR request_id=$4) AND ($5='' OR source=$5) AND ($6='' OR ($6='matched' AND matched=TRUE AND status_code BETWEEN 200 AND 299) OR ($6='unmatched' AND matched=FALSE AND status_code BETWEEN 200 AND 299) OR ($6='error' AND status_code>=400)) AND ($7::timestamptz IS NULL OR created_at >= $7) AND ($8::timestamptz IS NULL OR created_at < $8)`
	args := []any{organizationID, projectID, strings.TrimSpace(filter.RulesetID), strings.TrimSpace(filter.RequestID), strings.TrimSpace(filter.Source), strings.TrimSpace(filter.Outcome), filter.Since, filter.Until}
	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invocation_logs WHERE `+where, args...).Scan(&total); err != nil {
		return InvocationLogPage{}, err
	}
	query := `SELECT document FROM invocation_logs WHERE ` + where + ` ORDER BY created_at DESC,id DESC`
	if !filter.Unpaged {
		size := filter.Size
		if size < 1 {
			size = 20
		}
		if size > 100 {
			size = 100
		}
		page := filter.Page
		if page < 1 {
			page = 1
		}
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", size, (page-1)*size)
	} else {
		query += " LIMIT 10000"
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return InvocationLogPage{}, err
	}
	defer rows.Close()
	items := make([]domain.InvocationLog, 0)
	for rows.Next() {
		var document []byte
		var item domain.InvocationLog
		if err := rows.Scan(&document); err != nil {
			return InvocationLogPage{}, err
		}
		if err := json.Unmarshal(document, &item); err != nil {
			return InvocationLogPage{}, err
		}
		items = append(items, item)
	}
	return InvocationLogPage{Items: items, Total: total}, rows.Err()
}

func (r *PersistentInvocationLogRepository) DeleteBefore(ctx context.Context, before time.Time, batchSize int) (int, error) {
	db := currentDatabase()
	if db == nil {
		return 0, fmt.Errorf("PostgreSQL is not initialized")
	}
	if batchSize < 1 {
		batchSize = 500
	}
	result, err := db.ExecContext(ctx, `WITH expired AS (SELECT id FROM invocation_logs WHERE created_at < $1 ORDER BY created_at LIMIT $2) DELETE FROM invocation_logs AS logs USING expired WHERE logs.id=expired.id`, before, batchSize)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func (r *PersistentInvocationLogRepository) RecordCleanup(deleted int, err error) {
	r.cleanupMu.Lock()
	defer r.cleanupMu.Unlock()
	r.cleanup.LastRun = time.Now().UTC()
	r.cleanup.LastDeleted = deleted
	r.cleanup.TotalDeleted += int64(deleted)
	r.cleanup.LastError = ""
	if err != nil {
		r.cleanup.LastError = err.Error()
	}
}

func (r *PersistentInvocationLogRepository) CleanupStatus() InvocationCleanupState {
	r.cleanupMu.RLock()
	defer r.cleanupMu.RUnlock()
	return r.cleanup
}

func (r *PersistentInvocationLogRepository) DeleteProject(organizationID, projectID string) error {
	db := currentDatabase()
	if db == nil {
		return fmt.Errorf("PostgreSQL is not initialized")
	}
	_, err := db.Exec(`DELETE FROM invocation_logs WHERE organization_id=$1 AND project_id=$2`, organizationID, projectID)
	return err
}
