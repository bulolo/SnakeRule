package repo

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

func LoadEntity(key string, target any) error { return loadEntity(key, target) }
func SaveEntity(key string, value any) error  { return saveEntity(key, value) }

func loadEntity(key string, target any) error {
	db := currentDatabase()
	if db == nil {
		return errors.New("PostgreSQL must be initialized before loading data")
	}
	data, found, err := readEntity(db, key)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return json.Unmarshal(data, target)
}

func saveEntity(key string, value any) error {
	db := currentDatabase()
	if db == nil {
		return errors.New("PostgreSQL must be initialized before saving data")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err = syncEntity(tx, key, string(data)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
