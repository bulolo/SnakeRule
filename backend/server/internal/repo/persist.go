package repo

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
)

var databaseMu sync.RWMutex
var database *sql.DB

func SetDatabase(db *sql.DB) {
	databaseMu.Lock()
	database = db
	databaseMu.Unlock()
}

func currentDatabase() *sql.DB {
	databaseMu.RLock()
	defer databaseMu.RUnlock()
	return database
}

func OpenDatabase(db *sql.DB) error {
	return runMigrations(db)
}

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationAdvisoryLockKey int64 = 0x736e616b6572756c // "snakerul"

func runMigrations(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for database migrations: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationAdvisoryLockKey); err != nil {
		return fmt.Errorf("lock database migrations: %w", err)
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationAdvisoryLockKey); err != nil {
			slog.Warn("failed to release database migration lock", "error", err)
		}
	}()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure database migration dialect: %w", err)
	}
	goose.SetBaseFS(migrationFiles)
	goose.SetTableName("snakerule_goose_db_version")
	defer func() {
		goose.SetBaseFS(nil)
		goose.SetTableName("goose_db_version")
	}()
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func LoadPersistent(path string, target any) error { return loadPersistent(path, target) }
func SavePersistent(path string, value any) error  { return savePersistent(path, value) }

func loadPersistent(path string, target any) error {
	if path == "" {
		return nil
	}
	if db := currentDatabase(); db != nil {
		data, found, err := readEntityBucket(db, filepath.Base(path))
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		return json.Unmarshal(data, target)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func savePersistent(path string, value any) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if db := currentDatabase(); db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err = syncEntityBucket(tx, filepath.Base(path), string(data)); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
